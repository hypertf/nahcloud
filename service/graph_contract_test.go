package service

import (
	"regexp"
	"sync"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

func graphProject(t *testing.T, svc *Service, suffix string) (*domain.OrganizationWithAPIKey, *domain.Project) {
	t.Helper()
	org, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "org-" + suffix, Name: "Org " + suffix})
	require.NoError(t, err)
	project, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{Slug: "project-" + suffix, Name: "Project" + suffix})
	require.NoError(t, err)
	return org, project
}

func graphSubnet(t *testing.T, svc *Service, project *domain.Project, name, region, cidr string) (*domain.Network, *domain.Subnet) {
	t.Helper()
	network, err := svc.CreateNetwork(project.ID, domain.CreateNetworkRequest{Name: name, Region: region})
	require.NoError(t, err)
	subnet, err := svc.CreateSubnet(network, domain.CreateSubnetRequest{Name: name, CIDR: cidr})
	require.NoError(t, err)
	return network, subnet
}

func TestSubnetPlacementValidationAndConcurrentOverlap(t *testing.T) {
	svc, _ := newTestService(t)
	_, project := graphProject(t, svc, "subnets")
	network, first := graphSubnet(t, svc, project, "east-a", domain.RegionUSEast1, "10.20.0.0/24")
	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{32}$`), first.ID)

	for _, cidr := range []string{"8.8.0.0/24", "10.20.0.1/24", "10.0.0.0/15", "10.0.0.0/29"} {
		_, err := svc.CreateSubnet(network, domain.CreateSubnetRequest{Name: "invalid-" + cidr, CIDR: cidr})
		require.True(t, domain.IsInvalidInput(err), "%s: %v", cidr, err)
	}
	_, err := svc.CreateSubnet(network, domain.CreateSubnetRequest{Name: "overlap", CIDR: "10.20.0.128/25"})
	require.True(t, domain.IsAlreadyExists(err) || domain.IsConflict(err), err)

	_, west := graphSubnet(t, svc, project, "west", domain.RegionUSWest1, "10.20.0.0/24")
	require.Equal(t, first.CIDR, west.CIDR, "the same CIDR is valid in a different region")

	otherNetwork, err := svc.CreateNetwork(project.ID, domain.CreateNetworkRequest{Name: "east-b", Region: domain.RegionUSEast1})
	require.NoError(t, err)
	const contenders = 10
	results := make(chan error, contenders)
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, createErr := svc.CreateSubnet(otherNetwork, domain.CreateSubnetRequest{Name: "race", CIDR: "10.30.0.0/24"})
			results <- createErr
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for result := range results {
		if result == nil {
			succeeded++
		} else {
			require.True(t, domain.IsAlreadyExists(result) || domain.IsConflict(result), result)
		}
	}
	require.Equal(t, 1, succeeded)
}

func TestDiskAttachmentAndLoadBalancerContracts(t *testing.T) {
	svc, _ := newTestService(t)
	_, project := graphProject(t, svc, "compute")
	_, subnet := graphSubnet(t, svc, project, "apps", domain.RegionUSEast1, "10.40.0.0/24")
	instance, err := svc.CreateInstance(domain.CreateInstanceRequest{ProjectID: project.ID, Name: "app", Region: domain.RegionUSEast1, CPU: 1, MemoryMB: 512, Image: "image", Status: domain.StatusRunning, SubnetID: &subnet.ID})
	require.NoError(t, err)

	disk, err := svc.CreateDisk(project.ID, domain.CreateDiskRequest{Name: "data", Region: domain.RegionUSEast1, Type: "ssd", SizeGB: 10})
	require.NoError(t, err)
	tooSmall := 9
	_, err = svc.UpdateDisk(disk.ID, domain.UpdateDiskRequest{SizeGB: &tooSmall})
	require.True(t, domain.IsInvalidInput(err))
	larger := 20
	disk, err = svc.UpdateDisk(disk.ID, domain.UpdateDiskRequest{SizeGB: &larger})
	require.NoError(t, err)
	require.Equal(t, 20, disk.SizeGB)

	attachment, err := svc.CreateAttachment(disk, domain.CreateDiskAttachmentRequest{InstanceID: instance.ID, Device: "vdb"})
	require.NoError(t, err)
	_, err = svc.CreateAttachment(disk, domain.CreateDiskAttachmentRequest{InstanceID: instance.ID, Device: "vdc"})
	require.True(t, domain.IsAlreadyExists(err))
	require.True(t, domain.IsConflict(svc.DeleteDisk(disk.ID)))

	lb, err := svc.CreateLoadBalancer(project.ID, domain.CreateLoadBalancerRequest{Name: "frontend", SubnetID: subnet.ID, Protocol: "http", Port: 80, Algorithm: "round_robin", HealthCheckPath: "/health"})
	require.NoError(t, err)
	backend, err := svc.CreateBackend(lb, domain.CreateLoadBalancerBackendRequest{InstanceID: instance.ID, Port: 8080, Weight: 1})
	require.NoError(t, err)
	require.True(t, backend.Healthy)
	stopped := domain.StatusStopped
	_, err = svc.UpdateInstance(instance.ID, domain.UpdateInstanceRequest{Status: &stopped})
	require.NoError(t, err)
	backend, err = svc.GetBackend(backend.ID)
	require.NoError(t, err)
	require.False(t, backend.Healthy)
	require.True(t, domain.IsConflict(svc.DeleteSubnet(subnet.ID)))
	require.True(t, domain.IsConflict(svc.DeleteProject(project.ID)))

	require.NoError(t, svc.DeleteInstance(instance.ID))
	_, err = svc.GetAttachment(attachment.ID)
	require.True(t, domain.IsNotFound(err))
	_, err = svc.GetBackend(backend.ID)
	require.True(t, domain.IsNotFound(err))
	require.NoError(t, svc.DeleteLoadBalancer(lb.ID))
	require.NoError(t, svc.DeleteSubnet(subnet.ID))
}

func TestPolicyEvaluationDenyWinsAndCascades(t *testing.T) {
	svc, _ := newTestService(t)
	org, project := graphProject(t, svc, "policy")
	network, subnet := graphSubnet(t, svc, project, "policy-net", domain.RegionUSEast1, "10.50.0.0/24")

	allow, err := svc.CreatePolicy(org.ID, "", domain.CreatePolicyRequest{Name: "allow", Effect: "allow", Actions: []string{"*"}})
	require.NoError(t, err)
	allowBinding, err := svc.CreateBinding(allow.ID, domain.CreatePolicyBindingRequest{PrincipalType: "organization", PrincipalID: org.ID, TargetType: "organization", TargetID: org.ID})
	require.NoError(t, err)
	deny, err := svc.CreatePolicy(org.ID, "", domain.CreatePolicyRequest{Name: "deny", Effect: "deny", Actions: []string{"network.read"}})
	require.NoError(t, err)
	denyBinding, err := svc.CreateBinding(deny.ID, domain.CreatePolicyBindingRequest{PrincipalType: "organization", PrincipalID: org.ID, TargetType: "project", TargetID: project.ID})
	require.NoError(t, err)

	evaluation, err := svc.EvaluatePolicy(org.ID, domain.PolicyEvaluationRequest{PrincipalType: "organization", PrincipalID: org.ID, Action: "network.read", TargetType: "network", TargetID: network.ID})
	require.NoError(t, err)
	require.Equal(t, "deny", evaluation.Result)
	require.ElementsMatch(t, []string{allowBinding.ID, denyBinding.ID}, evaluation.MatchingBindingIDs)
	evaluation, err = svc.EvaluatePolicy(org.ID, domain.PolicyEvaluationRequest{PrincipalType: "organization", PrincipalID: org.ID, Action: "network.write", TargetType: "network", TargetID: network.ID})
	require.NoError(t, err)
	require.Equal(t, "allow", evaluation.Result)

	require.NoError(t, svc.DeleteSubnet(subnet.ID))
	require.NoError(t, svc.DeleteNetwork(network.ID))
	bindings, err := svc.ListBindings(deny.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 1, "project bindings survive deletion of an exact child target")
	require.NoError(t, svc.DeletePolicy(deny.ID))
	_, err = svc.GetBinding(denyBinding.ID)
	require.True(t, domain.IsNotFound(err))
}
