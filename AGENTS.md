# AGENTS.md

## Infrastructure

- Use Google Cloud services.
- Allow stateful workloads on Compute Engine.
- Prefer stateless deployment on Cloud Run where feasible, considering cost, scalability, and availability.
- Consider Cloud Storage (GCS) and Cloud SQL for persistent storage based on workload requirements.
- Prioritize cost efficiency, including free tiers and idle resource costs.
- Follow least-privilege IAM practices and use Secret Manager for sensitive data.
- Ensure persistent data is backed up and recoverable.
- Prefer reproducible infrastructure managed through Infrastructure as Code (IaC).
- Use Google Cloud's managed logging and monitoring services where practical.

## Markdown
- hard wrapしないこと
- Markdownファイルに対する編集は以下の状況でのみ許可されます:
    - ソースコードなどの変更により矛盾が生じた場合
    - 人間からの指示があった場合

## git
- コミットする際はcoauthored-byを必ず記載すること
