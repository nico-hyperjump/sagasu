// Package e2e provides end-to-end tests with a large corpus and multiple queries.
package e2e

import (
	"fmt"
	"strings"

	"github.com/hyperjump/sagasu/internal/models"
)

// E2EDocument is a document entry in the E2E corpus (id, title, content).
type E2EDocument struct {
	ID      string
	Title   string
	Content string
}

// QueryTestCase defines a query and the document ID(s) that must appear in search results.
// At least one of ExpectedDocIDs must be present in the combined (non_semantic + semantic) results.
type QueryTestCase struct {
	Query          string
	ExpectedDocIDs []string
	Description    string
}

// Corpus holds documents and query test cases for E2E tests.
type Corpus struct {
	Documents   []E2EDocument
	TestCases   []QueryTestCase
	TotalDocs   int
	TotalQueries int
}

// BuildCorpus returns a corpus of 100 documents with varied content and multiple query test cases.
// Each document has a unique "signature" phrase so queries can assert the correct doc is returned.
func BuildCorpus() *Corpus {
	docs := buildDocuments(100)
	cases := buildQueryTestCases(docs)
	return &Corpus{
		Documents:    docs,
		TestCases:    cases,
		TotalDocs:    len(docs),
		TotalQueries: len(cases),
	}
}

func buildDocuments(n int) []E2EDocument {
	topics := []struct {
		title   string
		phrase  string
		content string
	}{
		{"Python Guide", "Python programming language", "Python is a high-level programming language. Python programming language is used for web development and data science."},
		{"Kubernetes Docs", "Kubernetes container orchestration", "Kubernetes is an open-source container orchestration platform. Kubernetes container orchestration automates deployment and scaling."},
		{"React Tutorial", "React hooks and components", "React is a JavaScript library. React hooks and components enable building user interfaces."},
		{"Go Language", "Go golang concurrency", "Go is a statically typed language. Go golang concurrency is achieved with goroutines and channels."},
		{"PostgreSQL Manual", "PostgreSQL relational database", "PostgreSQL is an advanced relational database. PostgreSQL relational database supports JSON and full-text search."},
		{"Docker Handbook", "Docker container images", "Docker enables building and shipping applications. Docker container images are portable across environments."},
		{"Machine Learning", "machine learning algorithms", "Machine learning is a subset of AI. Machine learning algorithms learn patterns from data."},
		{"Neural Networks", "neural network deep learning", "Neural networks are inspired by the brain. Neural network deep learning powers modern AI."},
		{"REST API Design", "REST API endpoints", "REST is an architectural style for APIs. REST API endpoints use HTTP methods and status codes."},
		{"GraphQL Overview", "GraphQL query language", "GraphQL is a query language for APIs. GraphQL query language lets clients request exactly what they need."},
		{"TypeScript Handbook", "TypeScript type system", "TypeScript adds static types to JavaScript. TypeScript type system catches errors at compile time."},
		{"Redis Cache", "Redis in-memory cache", "Redis is an in-memory data store. Redis in-memory cache is used for sessions and caching."},
		{"Elasticsearch Guide", "Elasticsearch full-text search", "Elasticsearch is a search and analytics engine. Elasticsearch full-text search scales horizontally."},
		{"AWS Lambda", "AWS Lambda serverless", "AWS Lambda runs code without servers. AWS Lambda serverless scales automatically."},
		{"Terraform IaC", "Terraform infrastructure as code", "Terraform manages cloud infrastructure. Terraform infrastructure as code is declarative."},
		{"Prometheus Metrics", "Prometheus monitoring metrics", "Prometheus is a monitoring system. Prometheus monitoring metrics are time-series based."},
		{"gRPC Overview", "gRPC remote procedure calls", "gRPC is a high-performance RPC framework. gRPC remote procedure calls use HTTP/2 and protobuf."},
		{"OAuth 2.0", "OAuth 2.0 authorization", "OAuth 2.0 is an authorization framework. OAuth 2.0 authorization enables secure delegated access."},
		{"JWT Tokens", "JWT JSON web tokens", "JWT is a compact token format. JWT JSON web tokens are used for authentication."},
		{"CI/CD Pipelines", "CI/CD continuous integration", "CI/CD automates build and deployment. CI/CD continuous integration runs tests on every commit."},
		{"Git Workflow", "Git version control", "Git is a distributed version control system. Git version control tracks changes in source code."},
		{"SQL Basics", "SQL structured query language", "SQL is used to manage relational data. SQL structured query language has SELECT INSERT UPDATE DELETE."},
		{"Microservices", "microservices architecture", "Microservices split an app into small services. Microservices architecture enables independent deployment."},
		{"Kafka Streams", "Apache Kafka streaming", "Apache Kafka is a distributed event stream platform. Apache Kafka streaming handles high throughput."},
		{"Nginx Config", "Nginx reverse proxy", "Nginx is a web server and reverse proxy. Nginx reverse proxy balances load and serves static files."},
		{"OOP Principles", "object-oriented programming", "OOP organizes code around objects. Object-oriented programming uses encapsulation and inheritance."},
		{"Functional Programming", "functional programming paradigm", "Functional programming treats computation as functions. Functional programming paradigm avoids mutable state."},
		{"Design Patterns", "design patterns software", "Design patterns are reusable solutions. Design patterns software includes Singleton and Factory."},
		{"API Versioning", "API versioning strategy", "API versioning allows backward compatibility. API versioning strategy can use URL or headers."},
		{"Database Indexing", "database indexing performance", "Indexes speed up queries. Database indexing performance is critical for large tables."},
		{"Cryptography Basics", "cryptography encryption decryption", "Cryptography secures data. Cryptography encryption decryption uses keys and algorithms."},
		{"HTTPS TLS", "HTTPS TLS SSL certificates", "HTTPS encrypts web traffic. HTTPS TLS SSL certificates verify identity."},
		{"Load Balancing", "load balancing high availability", "Load balancers distribute traffic. Load balancing high availability prevents single points of failure."},
		{"Caching Strategies", "caching strategy cache invalidation", "Caching improves performance. Caching strategy cache invalidation must be designed carefully."},
		{"Event Sourcing", "event sourcing CQRS", "Event sourcing stores state as events. Event sourcing CQRS separates read and write models."},
		{"Domain-Driven Design", "domain-driven design DDD", "DDD focuses on the business domain. Domain-driven design DDD uses aggregates and bounded contexts."},
		{"Agile Scrum", "Agile Scrum sprint", "Agile is an iterative approach. Agile Scrum sprint typically lasts two weeks."},
		{"Unit Testing", "unit testing mock", "Unit tests verify small units of code. Unit testing mock isolates dependencies."},
		{"Integration Testing", "integration testing E2E", "Integration tests verify components together. Integration testing E2E validates full flows."},
		{"Dependency Injection", "dependency injection DI", "DI provides dependencies from outside. Dependency injection DI improves testability."},
		{"Semantic Search", "semantic search embeddings", "Semantic search uses meaning not just keywords. Semantic search embeddings capture context."},
		{"Keyword Search", "keyword search full-text", "Keyword search matches terms. Keyword search full-text uses inverted indexes."},
		{"Hybrid Search", "hybrid search fusion", "Hybrid combines keyword and semantic. Hybrid search fusion improves recall."},
		{"Vector Database", "vector database similarity", "Vector DBs store embeddings. Vector database similarity uses cosine or dot product."},
		{"Embedding Models", "embedding models sentence", "Embeddings represent text as vectors. Embedding models sentence transform text to dense vectors."},
		{"Chunking Strategy", "chunking strategy overlap", "Chunking splits long documents. Chunking strategy overlap preserves context."},
		{"RAG Overview", "RAG retrieval augmented", "RAG combines retrieval and generation. RAG retrieval augmented grounds LLMs in documents."},
		{"LLM Fine-tuning", "LLM fine-tuning training", "Fine-tuning adapts pre-trained models. LLM fine-tuning training requires labeled data."},
		{"Prompt Engineering", "prompt engineering few-shot", "Prompts guide model behavior. Prompt engineering few-shot uses examples in the prompt."},
		{"OpenAPI Spec", "OpenAPI specification", "OpenAPI describes REST APIs. OpenAPI specification is machine-readable."},
		{"WebSocket Protocol", "WebSocket real-time", "WebSockets enable bidirectional communication. WebSocket real-time is used for chat and live updates."},
		{"Message Queue", "message queue asynchronous", "Message queues decouple producers and consumers. Message queue asynchronous enables scaling."},
		{"Rate Limiting", "rate limiting throttling", "Rate limiting protects APIs. Rate limiting throttling can be per-user or global."},
		{"Circuit Breaker", "circuit breaker resilience", "Circuit breaker stops cascading failures. Circuit breaker resilience pattern fails fast."},
		{"Feature Flags", "feature flags rollout", "Feature flags toggle functionality. Feature flags rollout allows gradual release."},
		{"A/B Testing", "A/B testing experiment", "A/B testing compares variants. A/B testing experiment uses statistical significance."},
		{"Logging Best Practices", "logging structured logs", "Structured logging aids debugging. Logging structured logs use JSON or key-value."},
		{"Distributed Tracing", "distributed tracing spans", "Tracing follows requests across services. Distributed tracing spans show latency breakdown."},
		{"Security Headers", "security headers CORS", "Security headers protect browsers. Security headers CORS control cross-origin requests."},
		{"Input Validation", "input validation sanitization", "Validation rejects bad input. Input validation sanitization prevents injection."},
		{"Password Hashing", "password hashing bcrypt", "Passwords must be hashed. Password hashing bcrypt is resistant to rainbow tables."},
		{"RBAC Permissions", "RBAC role-based access", "RBAC assigns permissions by role. RBAC role-based access control is common in enterprise."},
		{"Audit Logging", "audit logging compliance", "Audit logs record who did what. Audit logging compliance is required in regulated industries."},
		{"Backup Strategy", "backup strategy recovery", "Backups protect against data loss. Backup strategy recovery includes RTO and RPO."},
		{"Disaster Recovery", "disaster recovery DR", "DR plans restore after outages. Disaster recovery DR involves failover and runbooks."},
		{"Scaling Horizontal", "horizontal scaling sharding", "Horizontal scaling adds more nodes. Horizontal scaling sharding partitions data."},
		{"Vertical Scaling", "vertical scaling resources", "Vertical scaling adds CPU or memory. Vertical scaling resources have limits."},
		{"Cost Optimization", "cost optimization cloud", "Cloud costs can grow quickly. Cost optimization cloud uses reserved instances and spot."},
		{"Green Computing", "green computing sustainability", "Green computing reduces environmental impact. Green computing sustainability focuses on efficiency."},
		{"Accessibility", "accessibility WCAG", "Accessibility ensures inclusive design. Accessibility WCAG provides guidelines."},
		{"Internationalization", "internationalization i18n", "i18n supports multiple languages. Internationalization i18n covers locale and formatting."},
		{"Mobile First", "mobile first responsive", "Mobile first designs for small screens first. Mobile first responsive adapts to viewport."},
		{"Progressive Web App", "progressive web app PWA", "PWAs work offline. Progressive web app PWA uses service workers."},
		{"Server-Side Rendering", "server-side rendering SSR", "SSR renders HTML on the server. Server-side rendering SSR improves SEO."},
		{"Static Site Generation", "static site generation SSG", "SSG pre-renders pages at build time. Static site generation SSG is fast and cheap."},
		{"Edge Computing", "edge computing latency", "Edge runs code close to users. Edge computing latency reduces round-trip time."},
		{"Serverless Cold Start", "serverless cold start", "Cold start is the first request delay. Serverless cold start can be mitigated with provisioned concurrency."},
		{"Graph Database", "graph database Neo4j", "Graph DBs store nodes and edges. Graph database Neo4j is used for relationships."},
		{"Time-Series DB", "time-series database", "Time-series DBs optimize for metrics. Time-series database stores values by timestamp."},
		{"Document Store", "document store MongoDB", "Document stores use flexible schemas. Document store MongoDB stores BSON documents."},
		{"Key-Value Store", "key-value store", "Key-value stores are simple and fast. Key-value store is used for caching and sessions."},
		{"CAP Theorem", "CAP theorem consistency", "CAP says you cannot have all three. CAP theorem consistency availability partition tolerance."},
		{"ACID Transactions", "ACID transactions database", "ACID guarantees reliability. ACID transactions database ensure atomicity and isolation."},
		{"Eventually Consistent", "eventually consistent", "Eventually consistent systems converge. Eventually consistent is used in distributed systems."},
		{"CRDT Overview", "CRDT conflict-free", "CRDTs enable conflict-free replication. CRDT conflict-free replicated data types merge without coordination."},
		{"Zero Trust", "zero trust security", "Zero trust assumes breach. Zero trust security verifies every request."},
		{"Defense in Depth", "defense in depth layers", "Multiple layers improve security. Defense in depth layers include network app and data."},
		{"Penetration Testing", "penetration testing pentest", "Pentest simulates attacks. Penetration testing pentest finds vulnerabilities."},
		{"Code Review", "code review pull request", "Code review catches bugs early. Code review pull request is a best practice."},
		{"Documentation", "documentation API docs", "Good documentation helps adoption. Documentation API docs should be up to date."},
		{"Onboarding Guide", "onboarding guide new hires", "Onboarding helps new team members. Onboarding guide new hires covers setup and culture."},
		{"Incident Response", "incident response runbook", "Incidents need a clear process. Incident response runbook defines steps."},
		{"Post-Mortem", "post-mortem blameless", "Post-mortems learn from incidents. Post-mortem blameless focuses on systems not people."},
		{"SLO and SLI", "SLO SLI reliability", "SLOs define target reliability. SLO SLI reliability uses error budget."},
		{"Chaos Engineering", "chaos engineering resilience", "Chaos engineering tests resilience. Chaos engineering resilience uses fault injection."},
		{"Blue-Green Deployment", "blue-green deployment", "Blue-green reduces deployment risk. Blue-green deployment keeps two environments."},
		{"Canary Release", "canary release gradual", "Canary rolls out to a subset. Canary release gradual reduces blast radius."},
		{"Feature Branch", "feature branch workflow", "Feature branches isolate work. Feature branch workflow merges via PR."},
		{"Trunk-Based Development", "trunk-based development", "Trunk-based keeps main always releasable. Trunk-based development uses short-lived branches."},
		{"Refactoring", "refactoring code quality", "Refactoring improves structure. Refactoring code quality preserves behavior."},
		{"Technical Debt", "technical debt payoff", "Technical debt has interest. Technical debt payoff requires dedicated effort."},
		{"Code Coverage", "code coverage tests", "Coverage measures test extent. Code coverage tests should focus on critical paths."},
		{"Performance Profiling", "performance profiling", "Profiling finds bottlenecks. Performance profiling uses CPU and memory tools."},
		{"Memory Leak", "memory leak debugging", "Memory leaks grow over time. Memory leak debugging uses heap dumps."},
		{"Deadlock Detection", "deadlock detection concurrency", "Deadlocks freeze systems. Deadlock detection concurrency requires care with locks."},
		{"Async Programming", "async programming await", "Async avoids blocking. Async programming await is used in many languages."},
		{"Error Handling", "error handling retry", "Errors must be handled. Error handling retry uses backoff strategies."},
		{"Graceful Shutdown", "graceful shutdown signal", "Graceful shutdown drains connections. Graceful shutdown signal handles SIGTERM."},
		{"Health Check", "health check liveness", "Health checks indicate readiness. Health check liveness is used by orchestrators."},
		{"Config Management", "config management environment", "Config varies by environment. Config management environment uses 12-factor."},
		{"Secrets Management", "secrets management vault", "Secrets must not be in code. Secrets management vault encrypts and audits."},
		{"Infrastructure as Code", "infrastructure as code", "IaC defines infra in code. Infrastructure as code enables versioning."},
		{"GitOps Workflow", "GitOps workflow Argo", "GitOps uses Git as source of truth. GitOps workflow Argo syncs cluster state."},
		{"Container Registry", "container registry Docker Hub", "Registries store images. Container registry Docker Hub is widely used."},
		{"Image Scanning", "image scanning vulnerability", "Image scanning finds CVEs. Image scanning vulnerability is part of supply chain security."},
		{"Supply Chain Security", "supply chain security", "Supply chain attacks are rising. Supply chain security includes SBOM and signing."},
		{"Open Source License", "open source license", "Licenses have obligations. Open source license compliance is important."},
		{"API Gateway", "API gateway routing", "API gateways sit in front of services. API gateway routing and rate limiting are common."},
		{"Service Mesh", "service mesh Istio", "Service mesh manages service-to-service traffic. Service mesh Istio provides mTLS and observability."},
		{"mTLS Overview", "mTLS mutual TLS", "mTLS authenticates both sides. mTLS mutual TLS uses client certificates."},
		{"Zero Downtime", "zero downtime deployment", "Zero downtime avoids outages. Zero downtime deployment uses rolling or blue-green."},
		{"Database Migration", "database migration schema", "Migrations evolve schema. Database migration schema should be reversible when possible."},
		{"Feature Toggle", "feature toggle release", "Feature toggles decouple deploy from release. Feature toggle release allows instant rollback."},
		{"Observability", "observability metrics logs", "Observability is metrics logs traces. Observability metrics logs help debug production."},
		{"SRE Practices", "SRE site reliability", "SRE balances reliability and velocity. SRE site reliability engineering uses error budgets."},
		{"On-Call Rotation", "on-call rotation", "On-call ensures 24/7 coverage. On-call rotation should be fair and sustainable."},
		{"Documentation as Code", "documentation as code", "Docs live next to code. Documentation as code uses Markdown and generators."},
		{"API First", "API first design", "API first designs the contract first. API first design improves consistency."},
		{"Contract Testing", "contract testing consumer", "Contract tests verify API contracts. Contract testing consumer and provider align."},
		{"Smoke Test", "smoke test sanity", "Smoke tests verify basic functionality. Smoke test sanity runs after deployment."},
		{"Regression Test", "regression test suite", "Regression tests prevent re-introduced bugs. Regression test suite grows over time."},
		{"Load Test", "load test performance", "Load tests simulate traffic. Load test performance finds limits."},
		{"Fuzz Testing", "fuzz testing random", "Fuzz testing uses random input. Fuzz testing random finds edge cases."},
		{"Property-Based Testing", "property-based testing", "Property-based tests generate inputs. Property-based testing verifies invariants."},
	}

	out := make([]E2EDocument, 0, n)
	for i := 0; i < n && i < len(topics); i++ {
		t := topics[i]
		id := fmt.Sprintf("e2e-doc-%03d", i+1)
		out = append(out, E2EDocument{
			ID:      id,
			Title:   t.title,
			Content: t.content,
		})
	}
	// If we need more than len(topics), duplicate with different IDs
	for len(out) < n {
		i := len(out)
		t := topics[i%len(topics)]
		id := fmt.Sprintf("e2e-doc-%03d", i+1)
		out = append(out, E2EDocument{
			ID:      id,
			Title:   fmt.Sprintf("%s (%d)", t.title, i+1),
			Content: t.content,
		})
	}
	return out
}

func buildQueryTestCases(docs []E2EDocument) []QueryTestCase {
	if len(docs) == 0 {
		return nil
	}
	// Build cases from first 100 docs: each query targets a phrase that appears in one doc.
	phrases := []string{
		"Python programming", "Kubernetes container", "React hooks", "Go golang", "PostgreSQL relational",
		"Docker container", "machine learning", "neural network", "REST API", "GraphQL query",
		"TypeScript type", "Redis in-memory", "Elasticsearch full-text", "AWS Lambda", "Terraform infrastructure",
		"Prometheus monitoring", "gRPC remote", "OAuth 2.0", "JWT JSON", "CI/CD continuous",
		"Git version", "SQL structured", "microservices architecture", "Apache Kafka", "Nginx reverse",
		"object-oriented", "functional programming", "design patterns", "API versioning", "database indexing",
		"cryptography encryption", "HTTPS TLS", "load balancing", "caching strategy", "event sourcing",
		"domain-driven design", "Agile Scrum", "unit testing", "integration testing", "dependency injection",
		"semantic search", "keyword search", "hybrid search", "vector database", "embedding models",
		"chunking strategy", "RAG retrieval", "LLM fine-tuning", "prompt engineering", "OpenAPI specification",
	}
	var cases []QueryTestCase
	used := make(map[string]bool)
	for _, p := range phrases {
		// Find first doc that contains this phrase (in title or content)
		for _, d := range docs {
			if containsPhrase(d, p) && !used[d.ID] {
				cases = append(cases, QueryTestCase{
					Query:          p,
					ExpectedDocIDs: []string{d.ID},
					Description:   fmt.Sprintf("query %q should return doc %s", p, d.ID),
				})
				used[d.ID] = true
				break
			}
		}
	}
	return cases
}

func containsPhrase(d E2EDocument, phrase string) bool {
	return strings.Contains(d.Title, phrase) || strings.Contains(d.Content, phrase)
}

// ToDocumentInputs converts the corpus documents to models.DocumentInput for indexing.
func (c *Corpus) ToDocumentInputs() []*models.DocumentInput {
	out := make([]*models.DocumentInput, len(c.Documents))
	for i := range c.Documents {
		d := &c.Documents[i]
		out[i] = &models.DocumentInput{
			ID:      d.ID,
			Title:   d.Title,
			Content: d.Content,
		}
	}
	return out
}
