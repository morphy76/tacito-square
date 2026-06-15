# Some APIs examples

## Create LLM binding

```json
{
  "name": "gemini-2.5-flash-lite",
  "description": "QA Testing Agent template",
  "model": "gemini-2.5-flash-lite",
  "temperature": 0.7,
  "max_tokens": 2048,
  "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
  "credentials_secret": "A[...]Q"
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
  "content": "You are a helpful AI assistant which is expert of social psicology and provides existential answers which help the participants to evaluate their life critically."
}
```

## Create a new agent

```json
{
  "name": "alone-in-the-darkness",
  "description": "The sole agent in a single agent community",
  "role": "spoke",
  "brain": {
      "model": "gemini-2.5-flash-lite",
      "temperature": 0.7,
      "max_tokens": 512,
      "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
      "credentials_secret": "AQ.Ab8RN6IqH3fX5qh4SB3adNCA-J5DpzkYstV0oalqxV9spB6LQQ"
  },
  "short_term_memory": {
      "key_namespace": "agent:alone:in:the:darkness:short",
      "ttl_seconds": 3600
  },
  "long_term_memory": {
      "collection_name": "agent-alone-in-the-darkness-long",
      "vector_dimension": 1536
  },
  "skills": [
      "fa1bbc83-b999-4381-9e32-8f5abb8b3e7c",
      "6666571f-e0a3-42a4-87f5-25e7351f6875"
  ],
  "prompt_template": "721fcb8c-45e1-45f3-81d7-860d6dc25676",
  "mcp_clients": null,
}
```
```json
{
    "name": "answerer",
    "description": "This agent is in charge of synthesizing the conversatinal turn, consolidating the researcher's responses, and delivering a clear, structured, and comprehensive final answer on the subject.",
    "role": "spoke",
    "brain": {
        "model": "gemini-2.5-flash-lite",
        "temperature": 0.7,
        "max_tokens": 512,
        "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
        "credentials_secret": "AQ.Ab8RN6IqH3fX5qh4SB3adNCA-J5DpzkYstV0oalqxV9spB6LQQ"
    },
    "short_term_memory": {
        "key_namespace": "agent:answerer:short",
        "ttl_seconds": 3600
    },
    "long_term_memory": {
        "collection_name": "agent-answerer-long",
        "vector_dimension": 1536
    },
    "skills": [
        "fa1bbc83-b999-4381-9e32-8f5abb8b3e7c",
        "6666571f-e0a3-42a4-87f5-25e7351f6875"
    ],
    "prompt_template": "f8d36e8c-1b6e-40f5-bdee-ed460aaa11cd",
    "mcp_clients": null,
}
```
```json
{
    "name": "enquirer",
    "description": "This agent is in charge to think questions to ask to the researcher in order to improve the researcher awareness about an unknown subject.",
    "role": "spoke",
    "brain": {
        "model": "gemini-2.5-flash-lite",
        "temperature": 0.7,
        "max_tokens": 512,
        "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
        "credentials_secret": "AQ.Ab8RN6IqH3fX5qh4SB3adNCA-J5DpzkYstV0oalqxV9spB6LQQ"
    },
    "short_term_memory": {
        "key_namespace": "agent:enquirer:short",
        "ttl_seconds": 3600
    },
    "long_term_memory": {
        "collection_name": "agent-enquirer-long",
        "vector_dimension": 1536
    },
    "skills": [
        "fa1bbc83-b999-4381-9e32-8f5abb8b3e7c",
        "6666571f-e0a3-42a4-87f5-25e7351f6875"
    ],
    "prompt_template": "dc6c6f05-ffe8-4139-a245-f38fa516b317",
    "mcp_clients": null,
}
```
```json
    {
        "id": "a0aee24a-27c7-4bf8-b75d-9966c83d9f2f",
        "tenant_id": "acme.com",
        "name": "enquirer",
        "description": "This agent is in charge to think questions to ask to the researcher in order to improve the researcher awareness about an unknown subject.",
        "role": "spoke",
        "brain": {
            "model": "gemini-2.5-flash-lite",
            "temperature": 0.7,
            "max_tokens": 512,
            "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
            "credentials_secret": "AQ.Ab8RN6IqH3fX5qh4SB3adNCA-J5DpzkYstV0oalqxV9spB6LQQ"
        },
        "short_term_memory": {
            "key_namespace": "agent:enquirer:short",
            "ttl_seconds": 3600
        },
        "long_term_memory": {
            "collection_name": "agent-enquirer-long",
            "vector_dimension": 1536
        },
        "skills": [
            "fa1bbc83-b999-4381-9e32-8f5abb8b3e7c",
            "6666571f-e0a3-42a4-87f5-25e7351f6875"
        ],
        "prompt_template": "dc6c6f05-ffe8-4139-a245-f38fa516b317",
        "mcp_clients": null,
        "status": "running",
        "community_id": "2ec94f91-712c-4fbb-883c-c928aa44273d",
        "tier": "",
        "created_at": "2026-06-08T05:47:18.99391Z",
        "updated_at": "2026-06-15T21:27:06.989418Z"
    },
    {
        "id": "ab40c50d-fc31-4cd3-aaf4-e3c0936a9a68",
        "tenant_id": "acme.com",
        "name": "hub-agent",
        "description": "In a hub-spoke topology, this agent is the hub, receiving inputs from the outbound, internally delegate subagents to achieve a final answer and, then, export the outputs to the outbound.",
        "role": "hub",
        "brain": {
            "model": "gemini-2.5-flash-lite",
            "temperature": 0.7,
            "max_tokens": 512,
            "endpoint": "https://generativelanguage.googleapis.com/v1beta/openai/",
            "credentials_secret": "AQ.Ab8RN6IqH3fX5qh4SB3adNCA-J5DpzkYstV0oalqxV9spB6LQQ"
        },
        "short_term_memory": {
            "key_namespace": "agent:hub:short",
            "ttl_seconds": 3600
        },
        "long_term_memory": {
            "collection_name": "agent-hub-long",
            "vector_dimension": 1536
        },
        "skills": null,
        "prompt_template": "00000000-0000-0000-0000-000000000000",
        "mcp_clients": null,
        "status": "running",
        "community_id": "2ec94f91-712c-4fbb-883c-c928aa44273d",
        "tier": "",
        "created_at": "2026-06-08T05:54:26.056736Z",
        "updated_at": "2026-06-15T21:27:09.981037Z"
    }
```

## Community

```json
{
  "name": "multi-agent",
  "description": "This comunity is used to verify that multi agent commmunities operate as expected pasing through an hb agent",
  "topology": "hub-spoke",
  "configuration": {}
}
```
```json
{
  "name": "single-agent",
  "description": "This comunity is used to verify that single agent commmunities still operate as expected",
  "topology": "single-agent",
  "configuration": {}
}
```

## Conversational pattern

Start a new thread

```json
{
  "schema_ref": "urn:tacito:schema:conversational:start-thread:v1",
  "payload": {
    "thread_id": "test-20",
    "community_id": "2ec94f91-712c-4fbb-883c-c928aa44273d"
  }
}
```

Add a user message

```json
{
  "schema_ref": "urn:tacito:schema:conversational:add-user-message:v1",
  "payload": {
    "thread_id": "test-20",
    "community_id": "2ec94f91-712c-4fbb-883c-c928aa44273d",
    "message": "Hello, my name is Riccardo, can you help me with a friend?"
  }
}
```

End the thread

```json
{
  "schema_ref": "urn:tacito:schema:conversational:end-thread:v1",
  "payload": {
    "thread_id": "test-20",
    "community_id": "2ec94f91-712c-4fbb-883c-c928aa44273d"
  }
}
```
