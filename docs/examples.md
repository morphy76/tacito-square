# Create a new agent

```json
{
  "name": "qa-agent",
  "brain": {
    "model": "gemini-2.5-flash-lite",
    "temperature": 0.7,
    "max_tokens": 2048,
    "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
    "credentials_secret": "A[...]Q"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:qa:short"
  },
  "long_term_memory": {
    "collection_name": "agent-qa-long",
    "vector_dimension": 1536
  },
  "description": "QA Testing Agent template",
  "skills": [
    "1af8a9cd-da06-402e-beb9-95952534e6f1",
    "dce897a2-01b1-4ffc-afc2-b87c512b90cc"
  ],
  "prompt_template": "2abd89f1-e51d-461d-90a6-fbd2172a4f79",
  "mcp_clients": []
}

```