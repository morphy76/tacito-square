# Some APIs examples

brain: f84fe2b3-b6bf-4133-9935-18ed0c164357
skill off: 5e6e792b-8289-49ba-9eb2-4ba23b27cbc1
skill on: c027f061-7dec-4bcb-8e69-19fbbf85b521

prompt answer: 15b529e7-da0c-48ae-93e4-d49e0bf7f9c1
prompt enquirer: ba19e316-6a62-4467-92af-cc0e2fed17b6
prompt sys: c88fdaaf-c3d3-4419-83de-05f88b90a61c
prompt pessimistic: 40d4a2ee-c22b-4ab6-8356-424f326d6318
prompt optimistic: 186fe6cc-50c8-4d14-991e-c69b9fdf0a71

alone: 22a1f37c-e0fe-46bd-899e-c1d51804d5b4
answerer: aac51ddb-e09c-41ef-b74f-76e08355f8dd
enquirer: b6513d92-e98f-4bee-ad24-932bbb391fa4
orch: 980e1a05-9e0f-46a0-a242-0d35bd940f7d
optimistic: 38a3d373-1561-44cf-a7b2-e885f987d60c
pessimistic: 70411ae0-53e7-43fc-acfc-f8f898f59b68

single-agent: 7632e041-2e56-47ba-8b6e-ab56ee3eda63
multi: 8f559a1d-2de5-496f-bac7-bdc6ebff3daa

## Create LLM binding

> [!NOTE]
> The `api_key_secret_ref` field refers to an existing external Kubernetes Secret name in the target namespace (e.g., `tacito`). This secret must contain a key named `api-key` holding the actual API key value.
>
> To create this secret, run:
> ```bash
> kubectl create secret generic gemini-api-key --from-literal=api-key="YOUR_ACTUAL_API_KEY" -n tacito
> ```

```json
{
  "name": "The Brain - light",
  "provider": "openai",
  "api_base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
  "api_key_secret_ref": "gemini-api-key",
  "default_model": "gemini-2.5-flash-lite",
  "description": "Light brain for development purposes",
  "default_temperature": 0.7,
  "default_max_tokens": 512,
  "timeout_seconds": 30
}
```

## Skills

```json
{
  "name": "always-off",
  "description": "Never use this kill to process your answer",
  "content": "Remove punctuation from your final answer."
}
```
```json
{
  "name": "always-on",
  "description": "Always use this kill to process your answer.",
  "content": "Convert your final answer in uppercase."
}
```

## Prompt template

```json
{
  "name": "answer-behavior",
  "content": "You are the Synthesis Agent, a specialized cognitive agent responsible for compiling all insights, details, and answers gathered throughout the research conversation and presenting the final findings.\n\n### Core Objective\nAnalyze the conversation history, extract key themes, resolved queries, and critical insights, and produce a logical, objective, and comprehensive summary that serves as the final answer for the researcher.\n\n### Guidelines for Synthesis\n1. **Structured Layout:** Organize your output professionally using clear headings (e.g., Executive Summary, Key Findings/Themes, Resolution of Gaps, and Future Directions/Remaining Questions).\n2. **Grounded in History:** Base your final answer strictly on the facts, assertions, and reasoning captured in the thread. Do not introduce speculative information or external facts not discussed or referenced.\n3. **Clarity and Precision:** Translate complex dialogue segments into concise, actionable points. Eliminate redundancy and focus on high-value conclusions.\n4. **Highlight Resolved vs. Open Areas:** Clearly state what has been successfully clarified by the researcher and what remains unknown or requires further investigation.\n5. **Objective Tone:** Maintain an analytical, professional, and neutral tone throughout the final output.."
}
```
```json
{
  "name": "enquirer-behavior",
  "content": "You are the Inquiry Coach, a specialized cognitive agent whose sole objective is to help the researcher explore, unpack, and understand a complex or unknown subject. Your goal is not to supply answers, but to elevate the researcher's own awareness and uncover their blind spots.\n\n### Core Objective\nFormulate precise, thought-provoking questions that guide the researcher to articulate their understanding, identify assumptions, and clarify gaps in their knowledge.\n\n### Guidelines for Inquiry\n1. **Socratic Inquiry:** Ask open-ended, analytical, and probing questions rather than explaining concepts or giving advice.\n2. **Identify Blind Spots:** Ask about unstated assumptions, counter-arguments, dependencies, and potential points of failure within the researcher's current mental model.\n3. **Dynamically Adapt:** Active listening is key. Build your questions directly on the researcher's previous responses to guide them in a progressive line of thinking.\n4. **Explore Alternative Perspectives:** Prompt the researcher to consider the subject from different angles (e.g., structural, historical, systemic, or opposite viewpoints).\n5. **Simplicity and Focus:** Keep your prompts concise and ask only one or two focused questions at a time to keep the researcher's thinking structured and engaged."
}
```
```json
{
  "name": "system-behavior",
  "content": "You are a helpful AI assistant which is expert of social psicology and provides, in a clear and concise way, existential answers which help the participants to evaluate their life critically."
}
```
```json
{
  "name": "a-pessimistic-opinion",
  "content": "You always think that things will go wrong and you always try to find the worst possible outcome of every situation. You have an exact science of finding why everything will fail."
}
```
```json
{
  "name": "an-optimistic-opinion",
  "content": "You always think that things will go right and you always try to find the best possible outcome of every situation. You have an exact science of finding why everything will work."
}
```

## Create a new agent

```json
{
  "name": "alone-in-the-darkness",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:alone-in-the-darkness:short"
  },
  "long_term_memory": {
      "collection_name": "agent-alone-in-the-darkness-long",
      "vector_dimension": 1536
  },
  "description": "An agent that lives alone who is in charge of everything",
  "role": "spoke",
  "skills": [
    "5e6e792b-8289-49ba-9eb2-4ba23b27cbc1",
    "c027f061-7dec-4bcb-8e69-19fbbf85b521"
  ],
  "prompt_template": "c88fdaaf-c3d3-4419-83de-05f88b90a61c"
}
```
```json
{
  "name": "optimistic",
  "description": "This agent thinks that everyhing will work and always try to find the best possible outcome of every situation. He is the opposite of pessimistic. Enthusiastic.",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:optimistic:short"
  },
  "long_term_memory": {
      "collection_name": "agent-optimistic-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "186fe6cc-50c8-4d14-991e-c69b9fdf0a71"
}
```
```json
{
  "name": "pessimistic",
  "description": "This agent thinks that everyhing will fail and always try to find the worst possible outcome of every situation. He is the opposite of optimistic. Disenthusiastic.",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:pessimistic:short"
  },
  "long_term_memory": {
      "collection_name": "agent-pessimistic-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "40d4a2ee-c22b-4ab6-8356-424f326d6318"
}
```
```json
{
  "name": "answerer",
  "description": "This agent is in charge of synthesizing the conversatinal turn, consolidating the researcher's responses, and delivering a clear, structured, and comprehensive final answer on the subject.",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:answerer:short"
  },
  "long_term_memory": {
      "collection_name": "agent-answerer-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "15b529e7-da0c-48ae-93e4-d49e0bf7f9c1"
}
```
```json
{
  "name": "enquirer",
  "description": "This agent is in charge to think questions to ask to the researcher in order to improve the researcher awareness about an unknown subject.",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:enquirer:short"
  },
  "long_term_memory": {
      "collection_name": "agent-enquirer-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "skills": [
    "5e6e792b-8289-49ba-9eb2-4ba23b27cbc1",
    "c027f061-7dec-4bcb-8e69-19fbbf85b521"
  ],
  "prompt_template": "ba19e316-6a62-4467-92af-cc0e2fed17b6"
}
```
```json
{
  "name": "orchestrator",
  "description": "In a hub-spoke topology, this agent is the hub, receiving inputs from the outbound, internally delegate subagents to achieve a final answer and, then, export the outputs to the outbound.",
  "brain": {
    "llm_binding_id": "f84fe2b3-b6bf-4133-9935-18ed0c164357"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:orchestrator:short"
  },
  "long_term_memory": {
      "collection_name": "agent-orchestrator-long",
      "vector_dimension": 1536
  },
  "role": "hub",
  "skills": [
    "5e6e792b-8289-49ba-9eb2-4ba23b27cbc1",
    "c027f061-7dec-4bcb-8e69-19fbbf85b521"
  ],
  "prompt_template": "c88fdaaf-c3d3-4419-83de-05f88b90a61c"
}
```


## Community

```json
{
  "name": "multi-agent",
  "description": "This comunity is used to verify that multi agent commmunities operate as expected pasing through an hb agent",
  "topology": "hub-spoke"
}
```
```json
{
  "name": "single-agent",
  "description": "This comunity is used to verify that single agent commmunities still operate as expected",
  "topology": "single-agent"
}
```

## Conversational pattern

Start a new thread

```json
{
  "schema_ref": "urn:tacito:schema:conversational:start-thread:v1",
  "payload": {
    "thread_id": "test-5",
    "community_id": "405e81ab-fdba-4570-b0ac-b294cf616961"
  }
}
```

Add a user message

```json
{
  "schema_ref": "urn:tacito:schema:conversational:add-user-message:v1",
  "payload": {
    "thread_id": "test-5",
    "community_id": "405e81ab-fdba-4570-b0ac-b294cf616961",
    "message": "Hello, how are you?"
  }
}
```

```json
{
  "schema_ref": "urn:tacito:schema:conversational:end-thread:v1",
  "payload": {
    "thread_id": "test-5",
    "community_id": "405e81ab-fdba-4570-b0ac-b294cf616961"
  }
}
```
