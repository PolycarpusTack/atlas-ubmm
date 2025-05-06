# UBMM Deployment and Operations Guide

This guide provides detailed instructions for deploying, operating, and maintaining the Ultimate Backlog Management & Monitor (UBMM) system in various environments.

## Table of Contents

1. [Deployment Models](#deployment-models)
2. [AWS Deployment](#aws-deployment)
3. [Kubernetes Deployment](#kubernetes-deployment)
4. [Infrastructure Scaling](#infrastructure-scaling)
5. [Database Management](#database-management)
6. [Monitoring & Alerting](#monitoring--alerting)
7. [Backup & Recovery](#backup--recovery)
8. [Security Best Practices](#security-best-practices)
9. [Troubleshooting](#troubleshooting)
10. [Performance Optimization](#performance-optimization)

## Deployment Models

UBMM supports several deployment models depending on your organization's needs:

### Single-Tenant Cloud Deployment (Recommended)

- Dedicated infrastructure for each organization
- Complete isolation of data and resources
- Best for enterprise customers with specific compliance requirements

### Multi-Tenant Cloud Deployment

- Shared infrastructure with logical separation of data
- More cost-efficient for smaller organizations
- Standardized configuration across tenants

### On-Premises Deployment

- Deploy within your corporate network
- Full control over infrastructure and data
- Requires additional operational resources

## AWS Deployment

### Prerequisites

- AWS account with appropriate IAM permissions
- Terraform 1.8+ installed
- AWS CLI configured
- Access to AWS ECR for container images

### Deployment Steps

1. **Prepare Environment Variables**

   Create a `terraform.tfvars` file in the `infrastructure/terraform` directory:

   ```hcl
   environment        = "prod"
   name_prefix        = "ubmm"
   aws_region         = "eu-central-1"
   vpc_cidr           = "10.0.0.0/16"
   certificate_arn    = "arn:aws:acm:eu-central-1:1234567890:certificate/abcd1234-ef56-gh78-ij90-klmnopqrstuv"
   alarm_email_endpoints = ["alerts@example.com"]
   ```

2. **Initialize Terraform**

   ```bash
   cd infrastructure/terraform
   terraform init \
     -backend-config="bucket=ubmm-terraform-state" \
     -backend-config="key=prod/terraform.tfstate" \
     -backend-config="region=eu-central-1"
   ```

3. **Plan Deployment**

   ```bash
   terraform plan -var-file=terraform.tfvars -out=tfplan
   ```

4. **Apply Deployment**

   ```bash
   terraform apply tfplan
   ```

5. **Verify Deployment**

   ```bash
   # Check AWS resources
   aws ec2 describe-instances --filters "Name=tag:Project,Values=UBMM" --query "Reservations[].Instances[].{ID:InstanceId,State:State.Name,Type:InstanceType}"
   
   # Test load balancer endpoint
   curl -s $(terraform output -raw alb_dns_name)/health
   ```

### Infrastructure Components

The AWS deployment creates the following resources:

- **Networking**: VPC, subnets, route tables, NAT Gateways
- **Compute**: ECS Fargate clusters for containerized services
- **Database**: Aurora PostgreSQL Serverless v2
- **Caching**: ElastiCache Redis cluster
- **Messaging**: Amazon MSK (Managed Kafka)
- **Security**: Security groups, IAM roles, KMS keys
- **Load Balancing**: Application Load Balancer
- **Monitoring**: CloudWatch dashboards and alarms

## Kubernetes Deployment

UBMM can also be deployed on Kubernetes for organizations that prefer container orchestration.

### Prerequisites

- Kubernetes 1.24+ cluster
- kubectl configured
- Helm 3.x installed
- Container registry access

### Deployment Steps

1. **Configure Kubernetes Context**

   ```bash
   kubectl config use-context your-cluster-context
   ```

2. **Create Namespace**

   ```bash
   kubectl create namespace ubmm
   ```

3. **Deploy Database**

   ```bash
   helm repo add bitnami https://charts.bitnami.com/bitnami
   helm install postgres bitnami/postgresql -n ubmm \
     --set auth.username=ubmm \
     --set auth.password=your-secure-password \
     --set auth.database=ubmm \
     --set persistence.size=50Gi
   ```

4. **Deploy Redis**

   ```bash
   helm install redis bitnami/redis -n ubmm \
     --set auth.password=your-secure-password \
     --set master.persistence.size=10Gi
   ```

5. **Deploy Kafka**

   ```bash
   helm install kafka bitnami/kafka -n ubmm \
     --set persistence.size=50Gi \
     --set replicaCount=3
   ```

6. **Deploy UBMM Services**

   ```bash
   kubectl apply -f infrastructure/k8s/manifests/ -n ubmm
   ```

7. **Create Ingress**

   ```bash
   kubectl apply -f infrastructure/k8s/ingress.yaml -n ubmm
   ```

8. **Verify Deployment**

   ```bash
   kubectl get pods -n ubmm
   kubectl get svc -n ubmm
   kubectl get ingress -n ubmm
   ```

## Infrastructure Scaling

UBMM is designed to scale with your organization's needs.

### Vertical Scaling

| Component | Small (<50 users) | Medium (<200 users) | Large (>200 users) |
|-----------|-------------------|---------------------|-------------------|
| Database | 2 ACUs | 8 ACUs | 16+ ACUs |
| Redis | cache.t4g.small | cache.t4g.medium | cache.r6g.large |
| ECS Tasks | 0.5 vCPU, 1GB | 1 vCPU, 2GB | 2+ vCPU, 4+ GB |

### Horizontal Scaling

- **Frontend**: Configure auto-scaling based on CPU utilization (target: 70%)
- **API Gateway**: Scale based on concurrent connections (target: 1000 per instance)
- **Microservices**: Scale based on message queue depth and CPU utilization

### Example Auto-Scaling Configuration

```hcl
resource "aws_appautoscaling_target" "api_gateway" {
  max_capacity       = 10
  min_capacity       = 2
  resource_id        = "service/${aws_ecs_cluster.main.name}/${aws_ecs_service.api_gateway.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "api_gateway_cpu" {
  name               = "api-gateway-cpu"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.api_gateway.resource_id
  scalable_dimension = aws_appautoscaling_target.api_gateway.scalable_dimension
  service_namespace  = aws_appautoscaling_target.api_gateway.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value = 70.0
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}
```

## Database Management

### Migrations

UBMM uses a standard migration approach to manage database schema changes.

```bash
# Run migrations manually
cd services/backlog-service
go run cmd/migrate/main.go up

# Check migration status
go run cmd/migrate/main.go status
```

In production environments, migrations are applied automatically during deployment through the CI/CD pipeline.

### Backup Strategy

1. **Automated Backups**:
   - Aurora automatically creates daily backups with a 30-day retention period
   - Point-in-time recovery (PITR) is enabled

2. **Manual Snapshots**:
   - Create snapshots before major changes
   ```bash
   aws rds create-db-cluster-snapshot \
     --db-cluster-identifier ubmm-cluster-prod \
     --db-cluster-snapshot-identifier ubmm-pre-upgrade-snapshot
   ```

3. **Cross-Region Replication**:
   - For disaster recovery, enable cross-region replication
   ```bash
   aws rds create-db-cluster-parameter-group \
     --db-cluster-parameter-group-name ubmm-dr-params \
     --db-parameter-group-family aurora-postgresql15 \
     --description "DR parameter group"
   
   aws rds modify-db-cluster-parameter-group \
     --db-cluster-parameter-group-name ubmm-dr-params \
     --parameters "ParameterName=rds.logical_replication,ParameterValue=1,ApplyMethod=pending-reboot"
   ```

### Monitoring Database Performance

Monitor these key metrics for optimal database performance:

- CPU Utilization (target: <70%)
- Freeable Memory (target: >25%)
- Read/Write IOPS (baseline understanding)
- Connection count (target: <80% of max connections)
- Buffer cache hit ratio (target: >95%)

## Monitoring & Alerting

UBMM leverages a comprehensive monitoring stack to ensure system health and performance.

### Metrics Collection

- **System Metrics**: Collected via CloudWatch agents
- **Application Metrics**: Exposed via Prometheus endpoints
- **Business Metrics**: Custom CloudWatch metrics

### Alert Thresholds

| Metric | Warning | Critical | Description |
|--------|---------|----------|-------------|
| API Response Time | >200ms | >500ms | 95th percentile response time |
| Error Rate | >1% | >5% | Percentage of 5xx responses |
| CPU Utilization | >70% | >90% | Sustained for 5 minutes |
| Memory Usage | >80% | >95% | Sustained for 5 minutes |
| Disk Usage | >80% | >90% | Available disk space |
| Database Connections | >80% | >90% | Of maximum connections |

### Logging Strategy

UBMM uses a structured logging approach:

1. **Application Logs**: JSON-formatted logs with consistent fields
2. **Log Aggregation**: CloudWatch Logs for centralized storage
3. **Log Retention**: 30 days in hot storage, archived to S3 for long-term storage
4. **Log Analysis**: CloudWatch Logs Insights for ad-hoc queries

### Sample Log Query

```
filter @type = "APPLICATION" 
  and service = "backlog-service"
  and level = "ERROR"
| stats count(*) as errorCount by operation, errorType
| sort errorCount desc
| limit 20
```

## Backup & Recovery

### Backup Components

1. **Database**: Aurora automated backups and snapshots
2. **Configuration**: Terraform state in S3 with versioning
3. **Encryption Keys**: KMS key backups
4. **Application State**: Event sourcing provides full state reconstruction

### Disaster Recovery Procedure

1. **Assess the Incident**:
   - Determine scope and impact
   - Document current state

2. **Restore Infrastructure**:
   ```bash
   # Restore infrastructure in DR region
   cd infrastructure/terraform
   terraform init -backend-config="region=eu-west-1"
   terraform apply -var="environment=prod" -var="aws_region=eu-west-1"
   ```

3. **Restore Database**:
   ```bash
   # Restore from latest snapshot
   aws rds restore-db-cluster-from-snapshot \
     --db-cluster-identifier ubmm-cluster-prod-restored \
     --snapshot-identifier ubmm-latest-snapshot \
     --engine aurora-postgresql
   ```

4. **Verify Restoration**:
   - Run health checks against restored services
   - Verify data integrity

5. **Switch Traffic**:
   - Update DNS records
   - Monitor for issues during transition

## Security Best Practices

### Network Security

- **VPC Isolation**: Services in private subnets
- **Security Groups**: Least privilege access
- **WAF Rules**: Protection against common web threats
- **Network ACLs**: Additional subnet-level controls

### Data Security

- **Encryption at Rest**: All data stores use AWS KMS
- **Encryption in Transit**: TLS 1.3 for all communications
- **Data Classification**: PII and sensitive data marked and tracked
- **Data Retention**: Automated cleanup of stale data

### Access Control

- **IAM Roles**: Service roles with minimal permissions
- **Secrets Management**: AWS Secrets Manager for credentials
- **MFA**: Required for all human access to production
- **Role-Based Access**: Application-level permissions

### Compliance Automation

- **Automated Scanning**: Infrastructure-as-code security checks
- **Compliance Controls**: Mapped to SOC 2 requirements
- **Audit Logs**: Immutable logs for all control plane actions
- **Regular Reviews**: Quarterly access reviews

## Troubleshooting

### Common Issues

#### Service Unavailability

1. **Check ECS Service Status**:
   ```bash
   aws ecs describe-services \
     --cluster ubmm-cluster-prod \
     --services ubmm-api-gateway-prod
   ```

2. **Check Service Logs**:
   ```bash
   aws logs get-log-events \
     --log-group-name /ecs/ubmm-api-gateway-prod \
     --log-stream-name $(aws logs describe-log-streams --log-group-name /ecs/ubmm-api-gateway-prod --order-by LastEventTime --descending --limit 1 --query 'logStreams[0].logStreamName' --output text)
   ```

3. **Check Target Group Health**:
   ```bash
   aws elbv2 describe-target-health \
     --target-group-arn $(aws elbv2 describe-target-groups --query 'TargetGroups[?TargetGroupName==`ubmm-api-gateway-prod`].TargetGroupArn' --output text)
   ```

#### Database Performance Issues

1. **Check Performance Insights**:
   ```bash
   aws pi get-resource-metrics \
     --service-type RDS \
     --identifier $(aws rds describe-db-clusters --db-cluster-identifier ubmm-cluster-prod --query 'DBClusters[0].DBClusterIdentifier' --output text) \
     --metric-queries '{"Metric":"db.load.avg","GroupBy":{"Group":"db.sql","Limit":10}}' \
     --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
     --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
     --period-in-seconds 60
   ```

2. **Check Connection Count**:
   ```sql
   SELECT 
     count(*) as connection_count, 
     state, 
     usename 
   FROM 
     pg_stat_activity 
   GROUP BY 
     state, usename 
   ORDER BY 
     connection_count DESC;
   ```

### Debugging Steps

1. **Service Health Check**:
   ```bash
   # Check service endpoints
   curl -s https://api.ubmm.example.com/health | jq
   
   # Check component health
   for svc in backlog-service api-gateway frontend; do
     echo "Checking $svc..."
     curl -s https://api.ubmm.example.com/health/$svc | jq
   done
   ```

2. **Distributed Tracing**:
   - Open Tempo UI at https://observability.ubmm.example.com/tempo
   - Search for recent traces with errors
   - Analyze service dependencies and latencies

3. **Log Analysis**:
   - Search for error patterns across services
   - Correlate errors with deployment or configuration changes
   - Check for resource constraints

## Performance Optimization

### Caching Strategy

UBMM uses a multi-level caching approach:

1. **Browser Caching**: Static assets with appropriate Cache-Control headers
2. **CDN Caching**: CloudFront for global content delivery
3. **Application Caching**: Redis for API responses and computed values
4. **Database Caching**: Connection pooling and query result caching

### Cache Configuration

```yaml
# Redis cache TTLs by resource type
backlog_items: 300  # 5 minutes
backlog_metrics: 60  # 1 minute
user_preferences: 1800  # 30 minutes
workshop_templates: 86400  # 24 hours

# Cache invalidation patterns
item_update: ["backlog_items:*", "backlog_metrics"]
metrics_update: ["backlog_metrics"]
```

### Database Optimization

1. **Index Optimization**:
   ```sql
   -- Add index on commonly queried fields
   CREATE INDEX idx_backlog_items_status_priority ON backlog_items(status, priority);
   
   -- Add partial index for active items
   CREATE INDEX idx_backlog_items_active ON backlog_items(priority)
   WHERE status != 'DONE';
   ```

2. **Query Optimization**:
   ```sql
   -- Before: Slow query
   SELECT * FROM backlog_items WHERE status = 'NEW' ORDER BY priority ASC;
   
   -- After: Optimized with index
   SELECT id, title, description, status, priority 
   FROM backlog_items 
   WHERE status = 'NEW' 
   ORDER BY priority ASC
   LIMIT 50;
   ```

3. **Connection Pooling**:
   ```go
   // Configure optimal connection pool
   db.SetMaxOpenConns(25)
   db.SetMaxIdleConns(10)
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

### Load Testing

Regular load testing ensures the system can handle peak traffic:

```bash
# Run load test with k6
k6 run -e ENV=prod -e TARGET_URL=https://api.ubmm.example.com tests/performance/api_load_test.js

# Run database load test
pgbench -h ubmm-cluster-prod.cluster-xyz.eu-central-1.rds.amazonaws.com -U benchuser -d ubmm -c 20 -j 4 -T 60
```

Monitor key metrics during load tests:

- Response time (p50, p95, p99)
- Error rate
- CPU and memory usage
- Database connection count and query time
- Cache hit ratio

## Conclusion

This deployment and operations guide provides a comprehensive overview of UBMM infrastructure management. For additional assistance, contact the UBMM support team at [support@ubmm.example.com](mailto:support@ubmm.example.com).
