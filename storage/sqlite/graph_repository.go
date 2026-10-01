package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

type GraphRepository struct{ db *DB }

func NewGraphRepository(db *DB) *GraphRepository { return &GraphRepository{db: db} }

func graphWriteError(resource, name string, err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.AlreadyExistsError(resource, "identity", name)
	}
	if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		return domain.ForeignKeyViolationError(resource, "parent", name)
	}
	return fmt.Errorf("failed to write %s: %w", resource, err)
}

func (r *GraphRepository) CreateNetwork(v *domain.Network) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO networks(id,project_id,name,cidr,created_at,updated_at) VALUES(?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.CIDR, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("network", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) GetNetwork(id string) (*domain.Network, error) {
	v := new(domain.Network)
	err := r.db.QueryRow(`SELECT id,project_id,name,cidr,created_at,updated_at FROM networks WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("network", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListNetworks(projectID string) ([]*domain.Network, error) {
	rows, err := r.db.Query(`SELECT id,project_id,name,cidr,created_at,updated_at FROM networks WHERE project_id=? ORDER BY name,id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Network
	for rows.Next() {
		v := new(domain.Network)
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateNetwork(v *domain.Network) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE networks SET name=?,cidr=?,updated_at=? WHERE id=?`, v.Name, v.CIDR, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("network", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) DeleteNetwork(id string) error {
	_, err := r.db.Exec(`DELETE FROM networks WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateSubnet(v *domain.Subnet) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO subnets(id,network_id,project_id,name,cidr,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.NetworkID, v.ProjectID, v.Name, v.CIDR, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("subnet", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) GetSubnet(id string) (*domain.Subnet, error) {
	v := new(domain.Subnet)
	err := r.db.QueryRow(`SELECT id,network_id,project_id,name,cidr,created_at,updated_at FROM subnets WHERE id=?`, id).Scan(&v.ID, &v.NetworkID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("subnet", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListSubnets(parent string) ([]*domain.Subnet, error) {
	rows, err := r.db.Query(`SELECT id,network_id,project_id,name,cidr,created_at,updated_at FROM subnets WHERE network_id=? ORDER BY name,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Subnet
	for rows.Next() {
		v := new(domain.Subnet)
		if err := rows.Scan(&v.ID, &v.NetworkID, &v.ProjectID, &v.Name, &v.CIDR, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateSubnet(v *domain.Subnet) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE subnets SET name=?,cidr=?,updated_at=? WHERE id=?`, v.Name, v.CIDR, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("subnet", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) DeleteSubnet(id string) error {
	_, err := r.db.Exec(`DELETE FROM subnets WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateDisk(v *domain.Disk) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO disks(id,project_id,name,size_gb,created_at,updated_at) VALUES(?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.SizeGB, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("disk", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) GetDisk(id string) (*domain.Disk, error) {
	v := new(domain.Disk)
	err := r.db.QueryRow(`SELECT id,project_id,name,size_gb,created_at,updated_at FROM disks WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.SizeGB, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("disk", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListDisks(parent string) ([]*domain.Disk, error) {
	rows, err := r.db.Query(`SELECT id,project_id,name,size_gb,created_at,updated_at FROM disks WHERE project_id=? ORDER BY name,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Disk
	for rows.Next() {
		v := new(domain.Disk)
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.SizeGB, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateDisk(v *domain.Disk) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE disks SET name=?,size_gb=?,updated_at=? WHERE id=?`, v.Name, v.SizeGB, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("disk", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) DeleteDisk(id string) error {
	_, err := r.db.Exec(`DELETE FROM disks WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateAttachment(v *domain.DiskAttachment) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO disk_attachments(id,disk_id,project_id,instance_id,device,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.DiskID, v.ProjectID, v.InstanceID, v.Device, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("disk attachment", v.Device, err)
	}
	return nil
}
func (r *GraphRepository) GetAttachment(id string) (*domain.DiskAttachment, error) {
	v := new(domain.DiskAttachment)
	err := r.db.QueryRow(`SELECT id,disk_id,project_id,instance_id,device,created_at,updated_at FROM disk_attachments WHERE id=?`, id).Scan(&v.ID, &v.DiskID, &v.ProjectID, &v.InstanceID, &v.Device, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("disk attachment", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListAttachments(parent string) ([]*domain.DiskAttachment, error) {
	rows, err := r.db.Query(`SELECT id,disk_id,project_id,instance_id,device,created_at,updated_at FROM disk_attachments WHERE disk_id=? ORDER BY device,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.DiskAttachment
	for rows.Next() {
		v := new(domain.DiskAttachment)
		if err := rows.Scan(&v.ID, &v.DiskID, &v.ProjectID, &v.InstanceID, &v.Device, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateAttachment(v *domain.DiskAttachment) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE disk_attachments SET device=?,updated_at=? WHERE id=?`, v.Device, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("disk attachment", v.Device, err)
	}
	return nil
}
func (r *GraphRepository) DeleteAttachment(id string) error {
	_, err := r.db.Exec(`DELETE FROM disk_attachments WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreatePolicy(v *domain.Policy) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	var project any = v.ProjectID
	if v.ProjectID == "" {
		project = nil
	}
	_, err := r.db.Exec(`INSERT INTO policies(id,org_id,project_id,name,document,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.OrgID, project, v.Name, v.Document, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("policy", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) GetPolicy(id string) (*domain.Policy, error) {
	v := new(domain.Policy)
	var project sql.NullString
	err := r.db.QueryRow(`SELECT id,org_id,project_id,name,document,created_at,updated_at FROM policies WHERE id=?`, id).Scan(&v.ID, &v.OrgID, &project, &v.Name, &v.Document, &v.CreatedAt, &v.UpdatedAt)
	v.ProjectID = project.String
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("policy", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListPolicies(orgID, projectID string) ([]*domain.Policy, error) {
	query := `SELECT id,org_id,project_id,name,document,created_at,updated_at FROM policies WHERE org_id=? AND project_id IS NULL ORDER BY name,id`
	args := []any{orgID}
	if projectID != "" {
		query = `SELECT id,org_id,project_id,name,document,created_at,updated_at FROM policies WHERE org_id=? AND project_id=? ORDER BY name,id`
		args = append(args, projectID)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Policy
	for rows.Next() {
		v := new(domain.Policy)
		var project sql.NullString
		if err := rows.Scan(&v.ID, &v.OrgID, &project, &v.Name, &v.Document, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.ProjectID = project.String
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdatePolicy(v *domain.Policy) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE policies SET name=?,document=?,updated_at=? WHERE id=?`, v.Name, v.Document, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("policy", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) DeletePolicy(id string) error {
	_, err := r.db.Exec(`DELETE FROM policies WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateBinding(v *domain.PolicyBinding) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO policy_bindings(id,policy_id,principal,role,created_at,updated_at) VALUES(?,?,?,?,?,?)`, v.ID, v.PolicyID, v.Principal, v.Role, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("policy binding", v.Principal, err)
	}
	return nil
}
func (r *GraphRepository) GetBinding(id string) (*domain.PolicyBinding, error) {
	v := new(domain.PolicyBinding)
	err := r.db.QueryRow(`SELECT id,policy_id,principal,role,created_at,updated_at FROM policy_bindings WHERE id=?`, id).Scan(&v.ID, &v.PolicyID, &v.Principal, &v.Role, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("policy binding", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListBindings(parent string) ([]*domain.PolicyBinding, error) {
	rows, err := r.db.Query(`SELECT id,policy_id,principal,role,created_at,updated_at FROM policy_bindings WHERE policy_id=? ORDER BY principal,role,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PolicyBinding
	for rows.Next() {
		v := new(domain.PolicyBinding)
		if err := rows.Scan(&v.ID, &v.PolicyID, &v.Principal, &v.Role, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateBinding(v *domain.PolicyBinding) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE policy_bindings SET principal=?,role=?,updated_at=? WHERE id=?`, v.Principal, v.Role, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("policy binding", v.Principal, err)
	}
	return nil
}
func (r *GraphRepository) DeleteBinding(id string) error {
	_, err := r.db.Exec(`DELETE FROM policy_bindings WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateLoadBalancer(v *domain.LoadBalancer) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO load_balancers(id,project_id,name,protocol,port,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.ProjectID, v.Name, v.Protocol, v.Port, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("load balancer", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) GetLoadBalancer(id string) (*domain.LoadBalancer, error) {
	v := new(domain.LoadBalancer)
	err := r.db.QueryRow(`SELECT id,project_id,name,protocol,port,created_at,updated_at FROM load_balancers WHERE id=?`, id).Scan(&v.ID, &v.ProjectID, &v.Name, &v.Protocol, &v.Port, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("load balancer", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListLoadBalancers(parent string) ([]*domain.LoadBalancer, error) {
	rows, err := r.db.Query(`SELECT id,project_id,name,protocol,port,created_at,updated_at FROM load_balancers WHERE project_id=? ORDER BY name,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LoadBalancer
	for rows.Next() {
		v := new(domain.LoadBalancer)
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.Protocol, &v.Port, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateLoadBalancer(v *domain.LoadBalancer) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE load_balancers SET name=?,protocol=?,port=?,updated_at=? WHERE id=?`, v.Name, v.Protocol, v.Port, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("load balancer", v.Name, err)
	}
	return nil
}
func (r *GraphRepository) DeleteLoadBalancer(id string) error {
	_, err := r.db.Exec(`DELETE FROM load_balancers WHERE id=?`, id)
	return err
}

func (r *GraphRepository) CreateBackend(v *domain.LoadBalancerBackend) error {
	v.CreatedAt, v.UpdatedAt = time.Now(), time.Now()
	_, err := r.db.Exec(`INSERT INTO load_balancer_backends(id,load_balancer_id,project_id,instance_id,port,weight,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.LoadBalancerID, v.ProjectID, v.InstanceID, v.Port, v.Weight, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return graphWriteError("load balancer backend", v.InstanceID, err)
	}
	return nil
}
func (r *GraphRepository) GetBackend(id string) (*domain.LoadBalancerBackend, error) {
	v := new(domain.LoadBalancerBackend)
	err := r.db.QueryRow(`SELECT id,load_balancer_id,project_id,instance_id,port,weight,created_at,updated_at FROM load_balancer_backends WHERE id=?`, id).Scan(&v.ID, &v.LoadBalancerID, &v.ProjectID, &v.InstanceID, &v.Port, &v.Weight, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("load balancer backend", id)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
func (r *GraphRepository) ListBackends(parent string) ([]*domain.LoadBalancerBackend, error) {
	rows, err := r.db.Query(`SELECT id,load_balancer_id,project_id,instance_id,port,weight,created_at,updated_at FROM load_balancer_backends WHERE load_balancer_id=? ORDER BY instance_id,port,id`, parent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LoadBalancerBackend
	for rows.Next() {
		v := new(domain.LoadBalancerBackend)
		if err := rows.Scan(&v.ID, &v.LoadBalancerID, &v.ProjectID, &v.InstanceID, &v.Port, &v.Weight, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *GraphRepository) UpdateBackend(v *domain.LoadBalancerBackend) error {
	v.UpdatedAt = time.Now()
	_, err := r.db.Exec(`UPDATE load_balancer_backends SET port=?,weight=?,updated_at=? WHERE id=?`, v.Port, v.Weight, v.UpdatedAt, v.ID)
	if err != nil {
		return graphWriteError("load balancer backend", v.InstanceID, err)
	}
	return nil
}
func (r *GraphRepository) DeleteBackend(id string) error {
	_, err := r.db.Exec(`DELETE FROM load_balancer_backends WHERE id=?`, id)
	return err
}
