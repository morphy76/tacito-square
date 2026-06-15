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
  "name": "qa-agent",
  "llm_binding_id": "99f09cc8-0bf4-48b4-83e5-8840f57a7777",
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
