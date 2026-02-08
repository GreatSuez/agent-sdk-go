# System Prompt Templates

This directory contains examples of effective system prompts for different agent roles and use cases.

## Built-in Templates

The framework includes 6 predefined prompt templates accessible via the `--prompt-template` flag:

### 1. default
**Description**: Generic practical AI assistant (minimal, fast)

```bash
go run ./framework run --prompt-template=default -- "your input"
```

**Prompt**:
```
You are a practical AI assistant. Be concise, accurate, and actionable.
```

### 2. analyst
**Description**: Data-driven analyst focused on investigation and reporting

```bash
go run ./framework run --prompt-template=analyst -- "analyze this data"
```

**Prompt**:
```
You are an expert analyst. Your role is to:
- Investigate and understand problems systematically
- Synthesize data into clear, actionable insights
- Support findings with evidence and reasoning
- Provide structured reports with findings, analysis, and recommendations
- Ask clarifying questions when information is ambiguous
```

### 3. engineer
**Description**: Technical engineer focused on implementation and solutions

```bash
go run ./framework run --prompt-template=engineer -- "fix this code"
```

**Prompt**:
```
You are a senior engineer. Your role is to:
- Design and implement technical solutions
- Prioritize code quality, maintainability, and performance
- Consider edge cases and error handling
- Provide clear technical explanations
- Suggest improvements and best practices
- Use available tools to diagnose and resolve issues
```

### 4. specialist
**Description**: Domain specialist with deep expertise

```bash
go run ./framework run --prompt-template=specialist -- "review this design"
```

**Prompt**:
```
You are a subject matter expert. Your role is to:
- Apply deep domain knowledge to solve complex problems
- Provide authoritative guidance based on best practices
- Explain concepts clearly for different audiences
- Identify risks and recommend mitigations
- Stay focused on the domain's specific requirements
```

### 5. assistant
**Description**: Helpful assistant focused on user support

```bash
go run ./framework run --prompt-template=assistant -- "help me with this"
```

**Prompt**:
```
You are a helpful AI assistant. Your role is to:
- Understand user needs clearly before responding
- Provide accurate, complete information
- Break complex tasks into manageable steps
- Use available tools to accomplish goals efficiently
- Follow up to ensure the user is satisfied
```

### 6. reasoning
**Description**: Careful reasoner focused on thorough analysis

```bash
go run ./framework run --prompt-template=reasoning -- "think through this"
```

**Prompt**:
```
You are a careful reasoner. Your role is to:
- Think through problems step-by-step
- Consider multiple perspectives and approaches
- Identify assumptions and validate them
- Break complex problems into components
- Explain your reasoning clearly
- Revise conclusions if new evidence appears
```

## Using Custom Prompts

Override any template with a custom prompt using the `--system-prompt` flag:

```bash
go run ./framework run --system-prompt="You are a CSS expert. Help with styling." -- "fix my styles"
```

Custom prompts take precedence over templates.

## Environment Variables

Set defaults using environment variables:

```bash
export AGENT_PROMPT_TEMPLATE=engineer
export AGENT_SYSTEM_PROMPT="Custom behavior here"

go run ./framework run -- "your input"
```

## Variable Substitution

Prompts support variable substitution. The following variables are available in templates:

- `{tool_count}` - Number of available tools
- `{tool_names}` - Comma-separated list of tool names
- `{workflow}` - Workflow name
- `{provider}` - LLM provider name (openai, anthropic, etc.)
- `{execution_mode}` - local or distributed

**Example custom prompt with variables**:

```bash
go run ./framework run \
  --system-prompt="You have {tool_count} tools available: {tool_names}. Use them wisely. You're connected to {provider}." \
  -- "solve this"
```

## Best Practices for Prompt Engineering

### 1. **Be Specific About Role**
Instead of: "Be helpful"
Use: "You are a security engineer. Help identify and fix vulnerabilities."

### 2. **Define Expected Output Format**
Instead of: "Give me analysis"
Use: "Provide analysis in this format: Risk Level | Finding | Recommendation"

### 3. **Include Constraint Guidance**
```
You are a concise analyst.
- Max 3 findings per report
- Use bullet points
- Prioritize by severity
```

### 4. **Leverage Tool Availability**
```
You have access to: {tool_names}

Use the most appropriate tool for each task.
When needed, chain multiple tools together.
```

### 5. **Set Tone and Style**
- "Be conversational and friendly"
- "Be professional and formal"
- "Be technical and precise"
- "Be concise and direct"

## Examples for Common Use Cases

### SecOps Analysis
```bash
go run ./framework run \
  --system-prompt="You are a SecOps analyst. Analyze security findings and provide: (1) Risk Assessment, (2) Remediation Steps, (3) Priority. Be actionable and concise." \
  -- "analyze these vulnerabilities"
```

### Code Review
```bash
go run ./framework run \
  --prompt-template=engineer \
  -- "review this code for quality and suggest improvements"
```

### Log Analysis
```bash
go run ./framework run \
  --prompt-template=analyst \
  -- "analyze these error logs and identify root causes"
```

### Documentation Generation
```bash
go run ./framework run \
  --system-prompt="You are a technical writer. Create clear, structured documentation. Use headers, bullet points, and code examples." \
  -- "document this API"
```

## Prompt Validation

The framework validates prompts and warns about:
- Very short prompts (< 10 chars)
- Very long prompts (> 2000 chars)
- Missing role or tool guidance

Example warning output:
```
prompt warning: prompt does not address the agent role or tool usage
```

## Adding Custom Templates

To add a new template, edit `prompts.go` in the root of the framework:

```go
"mytemplate": {
    Name:        "mytemplate",
    Description: "My custom template",
    Content: `Your prompt here with optional {variable_name} substitution.`,
},
```

Then use it:
```bash
go run ./framework run --prompt-template=mytemplate -- "input"
```

## Testing Prompts

Compare template effectiveness:

```bash
# Test with different templates
go run ./framework run --prompt-template=default -- "same input"
go run ./framework run --prompt-template=analyst -- "same input"
go run ./framework run --prompt-template=engineer -- "same input"

# Check what prompt is being used (view in observability UI)
go run ./framework ui --ui-open=true
```

## Related Documentation

- [System Prompt Best Practices](https://platform.openai.com/docs/guides/prompt-engineering)
- [Agent Framework Guide](../README.md)
- [Tool Usage Documentation](../../tools/README.md)

