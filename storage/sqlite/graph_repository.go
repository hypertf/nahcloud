package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

type GraphRepository struct{ db *DB }

func NewGraphRepository(db *DB) *GraphRepository { return &GraphRepository{db} }

func graphWriteError(resource, value string, err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.AlreadyExistsError(resource, "identity", value)
	}
	if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		return domain.ForeignKeyViolationError(resource, "parent", value)
	}
	return fmt.Errorf("write %s: %w", resource, err)
}
func times() (time.Time, time.Time) { now := time.Now().UTC(); return now, now }

func checkProjectNodeLimit(tx *sql.Tx, table, project string) error {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE project_id=?`, project).Scan(&count); err != nil {
		return err
	}
	if count >= 1000 {
		return domain.LimitExceededError("project resource limit exceeded")
	}
	return nil
}

func checkOrgEdgeLimit(tx *sql.Tx, org string) error {
	var count int
	err := tx.QueryRow(`SELECT
		(SELECT COUNT(*) FROM disk_attachments a JOIN projects p ON p.id=a.project_id WHERE p.org_id=?) +
		(SELECT COUNT(*) FROM policy_bindings WHERE org_id=?) +
		(SELECT COUNT(*) FROM load_balancer_backends b JOIN load_balancers l ON l.id=b.load_balancer_id JOIN projects p ON p.id=l.project_id WHERE p.org_id=?)`, org, org, org).Scan(&count)
	if err != nil {
		return err
	}
	if count >= 5000 {
		return domain.LimitExceededError("organization graph edge limit exceeded")
	}
	return nil
}

func networksOverlap(first, second *net.IPNet) bool {
	return first.Contains(second.IP) || second.Contains(first.IP)
}

func (r *GraphRepository) CreateNetwork(v *domain.Network) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = checkProjectNodeLimit(tx, "networks", v.ProjectID); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO networks(id,project_id,name,region,created_at,updated_at)VALUES(?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.Region, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("network", v.Name, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetNetwork(id string) (*domain.Network, error) {
	v := new(domain.Network)
	e := r.db.QueryRow(`SELECT id,project_id,name,region,created_at,updated_at FROM networks WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.Region, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("network", id)
	}
	return v, e
}
func (r *GraphRepository) ListNetworks(project string) (out []*domain.Network, e error) {
	out = []*domain.Network{}
	rows, e := r.db.Query(`SELECT id,project_id,name,region,created_at,updated_at FROM networks WHERE project_id=? ORDER BY name,id`, project)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.Network)
		if e = rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.Region, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateNetwork(v *domain.Network) error {
	v.UpdatedAt = time.Now().UTC()
	_, e := r.db.Exec(`UPDATE networks SET name=?,updated_at=? WHERE id=?`, v.Name, v.UpdatedAt, v.ID)
	if e != nil {
		return graphWriteError("network", v.Name, e)
	}
	return nil
}
func (r *GraphRepository) DeleteNetwork(id string) error {
	return r.deleteTarget("network", id, `DELETE FROM networks WHERE id=?`)
}

func (r *GraphRepository) CreateSubnet(v *domain.Subnet) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var region string
	if e = tx.QueryRow(`SELECT region FROM networks WHERE id=? AND project_id=?`, v.NetworkID, v.ProjectID).Scan(&region); e != nil {
		return domain.NotFoundError("network", v.NetworkID)
	}
	if e = checkProjectNodeLimit(tx, "subnets", v.ProjectID); e != nil {
		return e
	}
	_, candidate, e := net.ParseCIDR(v.CIDR)
	if e != nil {
		return domain.InvalidInputError("invalid CIDR", nil)
	}
	rows, e := tx.Query(`SELECT s.cidr FROM subnets s JOIN networks n ON n.id=s.network_id WHERE s.project_id=? AND n.region=?`, v.ProjectID, region)
	if e != nil {
		return e
	}
	for rows.Next() {
		var existingCIDR string
		if e = rows.Scan(&existingCIDR); e != nil {
			rows.Close()
			return e
		}
		_, existing, parseErr := net.ParseCIDR(existingCIDR)
		if parseErr == nil && networksOverlap(candidate, existing) {
			rows.Close()
			return domain.ConflictError("subnet CIDR overlaps another subnet in the project and region")
		}
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return e
	}
	if e = rows.Close(); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO subnets(id,network_id,project_id,name,cidr,created_at,updated_at)VALUES(?,?,?,?,?,?,?)`, v.ID, v.NetworkID, v.ProjectID, v.Name, v.CIDR, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("subnet", v.Name, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetSubnet(id string) (*domain.Subnet, error) {
	v := new(domain.Subnet)
	e := r.db.QueryRow(`SELECT id,network_id,project_id,name,cidr,created_at,updated_at FROM subnets WHERE id=?`, id).Scan(&v.ID, &v.NetworkID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("subnet", id)
	}
	return v, e
}
func (r *GraphRepository) ListSubnets(network string) (out []*domain.Subnet, e error) {
	out = []*domain.Subnet{}
	rows, e := r.db.Query(`SELECT id,network_id,project_id,name,cidr,created_at,updated_at FROM subnets WHERE network_id=? ORDER BY name,id`, network)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.Subnet)
		if e = rows.Scan(&v.ID, &v.NetworkID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) ListProjectSubnets(project string) (out []*domain.Subnet, e error) {
	out = []*domain.Subnet{}
	rows, e := r.db.Query(`SELECT id,network_id,project_id,name,cidr,created_at,updated_at FROM subnets WHERE project_id=? ORDER BY id`, project)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.Subnet)
		if e = rows.Scan(&v.ID, &v.NetworkID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateSubnet(v *domain.Subnet) error {
	v.UpdatedAt = time.Now().UTC()
	_, e := r.db.Exec(`UPDATE subnets SET name=?,updated_at=? WHERE id=?`, v.Name, v.UpdatedAt, v.ID)
	if e != nil {
		return graphWriteError("subnet", v.Name, e)
	}
	return nil
}
func (r *GraphRepository) DeleteSubnet(id string) error {
	return r.deleteTarget("subnet", id, `DELETE FROM subnets WHERE id=?`)
}

func (r *GraphRepository) CreateDisk(v *domain.Disk) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = checkProjectNodeLimit(tx, "disks", v.ProjectID); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO disks(id,project_id,name,region,type,size_gb,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.Region, v.Type, v.SizeGB, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("disk", v.Name, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetDisk(id string) (*domain.Disk, error) {
	v := new(domain.Disk)
	e := r.db.QueryRow(`SELECT id,project_id,name,region,type,size_gb,created_at,updated_at FROM disks WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.Region, &v.Type, &v.SizeGB, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("disk", id)
	}
	return v, e
}
func (r *GraphRepository) ListDisks(project string) (out []*domain.Disk, e error) {
	out = []*domain.Disk{}
	rows, e := r.db.Query(`SELECT id,project_id,name,region,type,size_gb,created_at,updated_at FROM disks WHERE project_id=? ORDER BY name,id`, project)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.Disk)
		if e = rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.Region, &v.Type, &v.SizeGB, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateDisk(v *domain.Disk) error {
	v.UpdatedAt = time.Now().UTC()
	result, e := r.db.Exec(`UPDATE disks SET name=?,size_gb=?,updated_at=? WHERE id=? AND size_gb<=?`, v.Name, v.SizeGB, v.UpdatedAt, v.ID, v.SizeGB)
	if e != nil {
		return graphWriteError("disk", v.Name, e)
	}
	updated, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if updated == 0 {
		return domain.InvalidInputError("disk size cannot be decreased", nil)
	}
	return nil
}
func (r *GraphRepository) DeleteDisk(id string) error {
	return r.deleteTarget("disk", id, `DELETE FROM disks WHERE id=?`)
}

func (r *GraphRepository) CreateAttachment(v *domain.DiskAttachment) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var dp, dr, ip, ir string
	e = tx.QueryRow(`SELECT project_id,region FROM disks WHERE id=?`, v.DiskID).Scan(&dp, &dr)
	if e != nil {
		return domain.NotFoundError("disk", v.DiskID)
	}
	e = tx.QueryRow(`SELECT project_id,region FROM instances WHERE id=?`, v.InstanceID).Scan(&ip, &ir)
	if e != nil || dp != ip || dr != ir {
		return domain.NotFoundError("instance", v.InstanceID)
	}
	var org string
	if e = tx.QueryRow(`SELECT org_id FROM projects WHERE id=?`, dp).Scan(&org); e != nil {
		return e
	}
	if e = checkOrgEdgeLimit(tx, org); e != nil {
		return e
	}
	v.ProjectID = dp
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO disk_attachments(id,disk_id,project_id,instance_id,device,created_at,updated_at)VALUES(?,?,?,?,?,?,?)`, v.ID, v.DiskID, v.ProjectID, v.InstanceID, v.Device, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("disk attachment", v.Device, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetAttachment(id string) (*domain.DiskAttachment, error) {
	v := new(domain.DiskAttachment)
	e := r.db.QueryRow(`SELECT id,disk_id,project_id,instance_id,device,created_at,updated_at FROM disk_attachments WHERE id=?`, id).Scan(&v.ID, &v.DiskID, &v.ProjectID, &v.InstanceID, &v.Device, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("disk attachment", id)
	}
	return v, e
}
func (r *GraphRepository) ListAttachments(disk string) (out []*domain.DiskAttachment, e error) {
	out = []*domain.DiskAttachment{}
	rows, e := r.db.Query(`SELECT id,disk_id,project_id,instance_id,device,created_at,updated_at FROM disk_attachments WHERE disk_id=? ORDER BY device,id`, disk)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.DiskAttachment)
		if e = rows.Scan(&v.ID, &v.DiskID, &v.ProjectID, &v.InstanceID, &v.Device, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) DeleteAttachment(id string) error {
	_, e := r.db.Exec(`DELETE FROM disk_attachments WHERE id=?`, id)
	return e
}

func (r *GraphRepository) CreatePolicy(v *domain.Policy) error {
	actions, _ := json.Marshal(v.Actions)
	v.CreatedAt, v.UpdatedAt = times()
	_, e := r.db.Exec(`INSERT INTO policies(id,org_id,name,description,effect,actions,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.OrgID, v.Name, v.Description, v.Effect, string(actions), v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("policy", v.Name, e)
	}
	return nil
}
func scanPolicy(row interface{ Scan(...any) error }) (*domain.Policy, error) {
	v := new(domain.Policy)
	var actions string
	e := row.Scan(&v.ID, &v.OrgID, &v.Name, &v.Description, &v.Effect, &actions, &v.CreatedAt, &v.UpdatedAt)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal([]byte(actions), &v.Actions)
	return v, e
}
func (r *GraphRepository) GetPolicy(id string) (*domain.Policy, error) {
	v, e := scanPolicy(r.db.QueryRow(`SELECT id,org_id,name,description,effect,actions,created_at,updated_at FROM policies WHERE id=?`, id))
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("policy", id)
	}
	return v, e
}
func (r *GraphRepository) ListPolicies(org, _ string) (out []*domain.Policy, e error) {
	out = []*domain.Policy{}
	rows, e := r.db.Query(`SELECT id,org_id,name,description,effect,actions,created_at,updated_at FROM policies WHERE org_id=? ORDER BY name,id`, org)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v, x := scanPolicy(rows)
		if x != nil {
			return nil, x
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdatePolicy(v *domain.Policy) error {
	actions, _ := json.Marshal(v.Actions)
	v.UpdatedAt = time.Now().UTC()
	_, e := r.db.Exec(`UPDATE policies SET name=?,description=?,effect=?,actions=?,updated_at=? WHERE id=?`, v.Name, v.Description, v.Effect, string(actions), v.UpdatedAt, v.ID)
	if e != nil {
		return graphWriteError("policy", v.Name, e)
	}
	return nil
}
func (r *GraphRepository) DeletePolicy(id string) error {
	_, e := r.db.Exec(`DELETE FROM policies WHERE id=?`, id)
	return e
}

func (r *GraphRepository) CreateBinding(v *domain.PolicyBinding) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var org string
	if e = tx.QueryRow(`SELECT org_id FROM policies WHERE id=?`, v.PolicyID).Scan(&org); e != nil {
		return domain.NotFoundError("policy", v.PolicyID)
	}
	if org != v.OrgID {
		return domain.NotFoundError("policy", v.PolicyID)
	}
	if e = validateBindingReference(tx, org, v.PrincipalType, v.PrincipalID, true); e != nil {
		return e
	}
	if e = validateBindingReference(tx, org, v.TargetType, v.TargetID, false); e != nil {
		return e
	}
	if e = checkOrgEdgeLimit(tx, org); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO policy_bindings(id,org_id,policy_id,principal_type,principal_id,target_type,target_id,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?,?)`, v.ID, v.OrgID, v.PolicyID, v.PrincipalType, v.PrincipalID, v.TargetType, v.TargetID, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("policy binding", v.ID, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetBinding(id string) (*domain.PolicyBinding, error) {
	v := new(domain.PolicyBinding)
	e := r.db.QueryRow(`SELECT id,org_id,policy_id,principal_type,principal_id,target_type,target_id,created_at,updated_at FROM policy_bindings WHERE id=?`, id).Scan(&v.ID, &v.OrgID, &v.PolicyID, &v.PrincipalType, &v.PrincipalID, &v.TargetType, &v.TargetID, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("policy binding", id)
	}
	return v, e
}
func (r *GraphRepository) ListBindings(policy string) (out []*domain.PolicyBinding, e error) {
	out = []*domain.PolicyBinding{}
	rows, e := r.db.Query(`SELECT id,org_id,policy_id,principal_type,principal_id,target_type,target_id,created_at,updated_at FROM policy_bindings WHERE policy_id=? ORDER BY id`, policy)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.PolicyBinding)
		if e = rows.Scan(&v.ID, &v.OrgID, &v.PolicyID, &v.PrincipalType, &v.PrincipalID, &v.TargetType, &v.TargetID, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) ListOrgBindings(org string) (out []*domain.PolicyBinding, e error) {
	out = []*domain.PolicyBinding{}
	rows, e := r.db.Query(`SELECT id,org_id,policy_id,principal_type,principal_id,target_type,target_id,created_at,updated_at FROM policy_bindings WHERE org_id=? ORDER BY id`, org)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.PolicyBinding)
		if e = rows.Scan(&v.ID, &v.OrgID, &v.PolicyID, &v.PrincipalType, &v.PrincipalID, &v.TargetType, &v.TargetID, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) DeleteBinding(id string) error {
	_, e := r.db.Exec(`DELETE FROM policy_bindings WHERE id=?`, id)
	return e
}

func (r *GraphRepository) CreateLoadBalancer(v *domain.LoadBalancer) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = tx.QueryRow(`SELECT n.region FROM subnets s JOIN networks n ON n.id=s.network_id WHERE s.id=? AND s.project_id=?`, v.SubnetID, v.ProjectID).Scan(&v.Region); e != nil {
		return domain.NotFoundError("subnet", v.SubnetID)
	}
	if e = checkProjectNodeLimit(tx, "load_balancers", v.ProjectID); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO load_balancers(id,project_id,name,subnet_id,region,protocol,port,algorithm,health_check_path,status,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.SubnetID, v.Region, v.Protocol, v.Port, v.Algorithm, v.HealthCheckPath, v.Status, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("load balancer", v.Name, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetLoadBalancer(id string) (*domain.LoadBalancer, error) {
	v := new(domain.LoadBalancer)
	e := r.db.QueryRow(`SELECT id,project_id,name,subnet_id,region,protocol,port,algorithm,health_check_path,status,created_at,updated_at FROM load_balancers WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.SubnetID, &v.Region, &v.Protocol, &v.Port, &v.Algorithm, &v.HealthCheckPath, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("load balancer", id)
	}
	return v, e
}
func (r *GraphRepository) ListLoadBalancers(project string) (out []*domain.LoadBalancer, e error) {
	out = []*domain.LoadBalancer{}
	rows, e := r.db.Query(`SELECT id,project_id,name,subnet_id,region,protocol,port,algorithm,health_check_path,status,created_at,updated_at FROM load_balancers WHERE project_id=? ORDER BY name,id`, project)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.LoadBalancer)
		if e = rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.SubnetID, &v.Region, &v.Protocol, &v.Port, &v.Algorithm, &v.HealthCheckPath, &v.Status, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateLoadBalancer(v *domain.LoadBalancer) error {
	v.UpdatedAt = time.Now().UTC()
	_, e := r.db.Exec(`UPDATE load_balancers SET name=?,algorithm=?,health_check_path=?,updated_at=? WHERE id=?`, v.Name, v.Algorithm, v.HealthCheckPath, v.UpdatedAt, v.ID)
	if e != nil {
		return graphWriteError("load balancer", v.Name, e)
	}
	return nil
}
func (r *GraphRepository) DeleteLoadBalancer(id string) error {
	return r.deleteTarget("load_balancer", id, `DELETE FROM load_balancers WHERE id=?`)
}

func (r *GraphRepository) CreateBackend(v *domain.LoadBalancerBackend) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var project, subnet string
	if e = tx.QueryRow(`SELECT project_id,subnet_id FROM load_balancers WHERE id=?`, v.LoadBalancerID).Scan(&project, &subnet); e != nil {
		return domain.NotFoundError("load balancer", v.LoadBalancerID)
	}
	var org string
	if e = tx.QueryRow(`SELECT org_id FROM projects WHERE id=?`, project).Scan(&org); e != nil {
		return e
	}
	var instanceProject string
	var instanceSubnet sql.NullString
	if e = tx.QueryRow(`SELECT project_id,subnet_id FROM instances WHERE id=?`, v.InstanceID).Scan(&instanceProject, &instanceSubnet); e != nil || instanceProject != project || !instanceSubnet.Valid || instanceSubnet.String != subnet {
		return domain.NotFoundError("instance", v.InstanceID)
	}
	if e = checkOrgEdgeLimit(tx, org); e != nil {
		return e
	}
	v.CreatedAt, v.UpdatedAt = times()
	_, e = tx.Exec(`INSERT INTO load_balancer_backends(id,load_balancer_id,instance_id,port,weight,enabled,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.LoadBalancerID, v.InstanceID, v.Port, v.Weight, v.Enabled, v.CreatedAt, v.UpdatedAt)
	if e != nil {
		return graphWriteError("load balancer backend", v.InstanceID, e)
	}
	return tx.Commit()
}
func (r *GraphRepository) GetBackend(id string) (*domain.LoadBalancerBackend, error) {
	v := new(domain.LoadBalancerBackend)
	var enabled int
	var status string
	e := r.db.QueryRow(`SELECT b.id,b.load_balancer_id,b.instance_id,b.port,b.weight,b.enabled,b.created_at,b.updated_at,i.status FROM load_balancer_backends b JOIN instances i ON i.id=b.instance_id WHERE b.id=?`, id).Scan(&v.ID, &v.LoadBalancerID, &v.InstanceID, &v.Port, &v.Weight, &enabled, &v.CreatedAt, &v.UpdatedAt, &status)
	if e == sql.ErrNoRows {
		return nil, domain.NotFoundError("load balancer backend", id)
	}
	v.Enabled = enabled != 0
	v.Healthy = v.Enabled && status == domain.StatusRunning
	return v, e
}
func (r *GraphRepository) ListBackends(lb string) (out []*domain.LoadBalancerBackend, e error) {
	out = []*domain.LoadBalancerBackend{}
	rows, e := r.db.Query(`SELECT b.id,b.load_balancer_id,b.instance_id,b.port,b.weight,b.enabled,b.created_at,b.updated_at,i.status FROM load_balancer_backends b JOIN instances i ON i.id=b.instance_id WHERE b.load_balancer_id=? ORDER BY b.instance_id,b.port,b.id`, lb)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		v := new(domain.LoadBalancerBackend)
		var enabled int
		var status string
		if e = rows.Scan(&v.ID, &v.LoadBalancerID, &v.InstanceID, &v.Port, &v.Weight, &enabled, &v.CreatedAt, &v.UpdatedAt, &status); e != nil {
			return
		}
		v.Enabled = enabled != 0
		v.Healthy = v.Enabled && status == domain.StatusRunning
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateBackend(v *domain.LoadBalancerBackend) error {
	v.UpdatedAt = time.Now().UTC()
	_, e := r.db.Exec(`UPDATE load_balancer_backends SET port=?,weight=?,enabled=?,updated_at=? WHERE id=?`, v.Port, v.Weight, v.Enabled, v.UpdatedAt, v.ID)
	if e != nil {
		return graphWriteError("load balancer backend", v.InstanceID, e)
	}
	return nil
}
func (r *GraphRepository) DeleteBackend(id string) error {
	_, e := r.db.Exec(`DELETE FROM load_balancer_backends WHERE id=?`, id)
	return e
}

func (r *GraphRepository) deleteTarget(target, id, statement string) error {
	tx, e := r.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`DELETE FROM policy_bindings WHERE target_type=? AND target_id=?`, target, id); e != nil {
		return e
	}
	res, e := tx.Exec(statement, id)
	if e != nil {
		if strings.Contains(e.Error(), "FOREIGN KEY constraint failed") {
			return domain.ConflictError(target + " is in use")
		}
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.NotFoundError(target, id)
	}
	return tx.Commit()
}

func validateBindingReference(tx *sql.Tx, org, kind, id string, principal bool) error {
	var query string
	if principal {
		switch kind {
		case "organization":
			query = `SELECT id FROM organizations WHERE id=? AND id=?`
		case "api_key":
			query = `SELECT id FROM api_keys WHERE id=? AND org_id=?`
		default:
			return domain.InvalidInputError("invalid principal_type", nil)
		}
	} else {
		switch kind {
		case "organization":
			query = `SELECT id FROM organizations WHERE id=? AND id=?`
		case "project":
			query = `SELECT id FROM projects WHERE id=? AND org_id=?`
		case "network":
			query = `SELECT n.id FROM networks n JOIN projects p ON p.id=n.project_id WHERE n.id=? AND p.org_id=?`
		case "subnet":
			query = `SELECT s.id FROM subnets s JOIN projects p ON p.id=s.project_id WHERE s.id=? AND p.org_id=?`
		case "instance":
			query = `SELECT i.id FROM instances i JOIN projects p ON p.id=i.project_id WHERE i.id=? AND p.org_id=?`
		case "disk":
			query = `SELECT d.id FROM disks d JOIN projects p ON p.id=d.project_id WHERE d.id=? AND p.org_id=?`
		case "load_balancer":
			query = `SELECT l.id FROM load_balancers l JOIN projects p ON p.id=l.project_id WHERE l.id=? AND p.org_id=?`
		case "bucket":
			query = `SELECT b.id FROM buckets b JOIN projects p ON p.id=b.project_id WHERE b.id=? AND p.org_id=?`
		default:
			return domain.InvalidInputError("invalid target_type", nil)
		}
	}
	var found string
	if e := tx.QueryRow(query, id, org).Scan(&found); e != nil {
		return domain.NotFoundError(kind, id)
	}
	return nil
}
