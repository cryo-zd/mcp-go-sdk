// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

// Protocol types for version 2025-06-18.
// To see the schema changes from the previous version, run:
//
//   prefix=https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/refs/heads/main/schema
//   sdiff -l <(curl $prefix/2025-03-26/schema.ts) <(curl $prefix/2025/06-18/schema.ts)

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
)

// Optional annotations for the client. The client can use annotations to inform
// how objects are used or displayed
type Annotations struct {
	// Describes who the intended customer of this object or data is.
	//
	// It can include multiple entries to indicate content useful for multiple
	// audiences (e.g., []Role{"user", "assistant"}).
	Audience []Role `json:"audience,omitempty"`
	// The moment the resource was last modified, as an ISO 8601 formatted string.
	//
	// Should be an ISO 8601 formatted string (e.g., "2025-01-12T15:00:58Z").
	//
	// Examples: last activity timestamp in an open file, timestamp when the
	// resource was attached, etc.
	LastModified string `json:"lastModified,omitempty"`
	// Describes how important this data is for operating the server.
	//
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that the
	// data is entirely optional.
	Priority float64 `json:"priority,omitempty"`
}

type CallToolParams = CallToolParamsFor[any]

type CallToolParamsFor[In any] struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta      `json:"_meta,omitempty"`
	Name      string `json:"name"`
	Arguments In     `json:"arguments,omitempty"`
}

// The server's response to a tool call.
type CallToolResult = CallToolResultFor[any]

type CallToolResultFor[Out any] struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// A list of content objects that represent the unstructured result of the tool
	// call.
	Content []Content `json:"content"`
	// An optional JSON object that represents the structured result of the tool
	// call.
	StructuredContent Out `json:"structuredContent,omitempty"`
	// Whether the tool call ended in an error.
	//
	// If not set, this is assumed to be false (the call was successful).
	//
	// Any errors that originate from the tool should be reported inside the result
	// object, with isError set to true, not as an MCP protocol-level error
	// response. Otherwise, the LLM would not be able to see that an error occurred
	// and self-correct.
	//
	// However, any errors in finding the tool, an error indicating that the
	// server does not support tool calls, or any other exceptional conditions,
	// should be reported as an MCP error response.
	IsError bool `json:"isError,omitempty"`
}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (x *CallToolResultFor[Out]) UnmarshalJSON(data []byte) error {
	type res CallToolResultFor[Out] // avoid recursion
	var wire struct {
		res
		Content []*wireContent `json:"content"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.res.Content, err = contentsFromWire(wire.Content, nil); err != nil {
		return err
	}
	*x = CallToolResultFor[Out](wire.res)
	return nil
}

func (x *CallToolParamsFor[Out]) GetProgressToken() any  { return getProgressToken(x) }
func (x *CallToolParamsFor[Out]) SetProgressToken(t any) { setProgressToken(x, t) }

type CancelledParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An optional string describing the reason for the cancellation. This may be
	// logged or presented to the user.
	Reason string `json:"reason,omitempty"`
	// The ID of the request to cancel.
	//
	// This must correspond to the ID of a request previously issued in the same
	// direction.
	RequestID any `json:"requestId"`
}

func (x *CancelledParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CancelledParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Capabilities a client may support. Known capabilities are defined here, in
// this schema, but this is not a closed set: any client can define its own,
// additional capabilities.
type ClientCapabilities struct {
	// Experimental, non-standard capabilities that the client supports.
	Experimental map[string]struct{} `json:"experimental,omitempty"`
	// Present if the client supports listing roots.
	Roots struct {
		// Whether the client supports notifications for changes to the roots list.
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
	// Present if the client supports sampling from an LLM.
	Sampling *SamplingCapabilities `json:"sampling,omitempty"`
	// Present if the client supports elicitation from the server.
	Elicitation *ElicitationCapabilities `json:"elicitation,omitempty"`
}

type CompleteParamsArgument struct {
	// The name of the argument
	Name string `json:"name"`
	// The value of the argument to use for completion matching.
	Value string `json:"value"`
}

// CompleteContext represents additional, optional context for completions.
type CompleteContext struct {
	// Previously-resolved variables in a URI template or prompt.
	Arguments map[string]string `json:"arguments,omitempty"`
}

// CompleteReference represents a completion reference type (ref/prompt ref/resource).
// The Type field determines which other fields are relevant.
type CompleteReference struct {
	Type string `json:"type"`
	// Name is relevant when Type is "ref/prompt".
	Name string `json:"name,omitempty"`
	// URI is relevant when Type is "ref/resource".
	URI string `json:"uri,omitempty"`
}

func (r *CompleteReference) UnmarshalJSON(data []byte) error {
	type wireCompleteReference CompleteReference // for naive unmarshaling
	var r2 wireCompleteReference
	if err := json.Unmarshal(data, &r2); err != nil {
		return err
	}
	switch r2.Type {
	case "ref/prompt", "ref/resource":
		if r2.Type == "ref/prompt" && r2.URI != "" {
			return fmt.Errorf("reference of type %q must not have a URI set", r2.Type)
		}
		if r2.Type == "ref/resource" && r2.Name != "" {
			return fmt.Errorf("reference of type %q must not have a Name set", r2.Type)
		}
	default:
		return fmt.Errorf("unrecognized content type %q", r2.Type)
	}
	*r = CompleteReference(r2)
	return nil
}

func (r *CompleteReference) MarshalJSON() ([]byte, error) {
	// Validation for marshalling: ensure consistency before converting to JSON.
	switch r.Type {
	case "ref/prompt":
		if r.URI != "" {
			return nil, fmt.Errorf("reference of type %q must not have a URI set for marshalling", r.Type)
		}
	case "ref/resource":
		if r.Name != "" {
			return nil, fmt.Errorf("reference of type %q must not have a Name set for marshalling", r.Type)
		}
	default:
		return nil, fmt.Errorf("unrecognized reference type %q for marshalling", r.Type)
	}

	type wireReference CompleteReference
	return json.Marshal(wireReference(*r))
}

type CompleteParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The argument's information
	Argument CompleteParamsArgument `json:"argument"`
	Context  *CompleteContext       `json:"context,omitempty"`
	Ref      *CompleteReference     `json:"ref"`
}

type CompletionResultDetails struct {
	HasMore bool     `json:"hasMore,omitempty"`
	Total   int      `json:"total,omitempty"`
	Values  []string `json:"values"`
}

// The server's response to a completion/complete request
type CompleteResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta       `json:"_meta,omitempty"`
	Completion CompletionResultDetails `json:"completion"`
}

type CreateMessageParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// A request to include context from one or more MCP servers (including the
	// caller), to be attached to the prompt. The client may ignore this request.
	IncludeContext string `json:"includeContext,omitempty"`
	// The maximum number of tokens to sample, as requested by the server. The
	// client may choose to sample fewer tokens than requested.
	MaxTokens int64              `json:"maxTokens"`
	Messages  []*SamplingMessage `json:"messages"`
	// Optional metadata to pass through to the LLM provider. The format of this
	// metadata is provider-specific.
	Metadata struct{} `json:"metadata,omitempty"`
	// The server's preferences for which model to select. The client may ignore
	// these preferences.
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	StopSequences    []string          `json:"stopSequences,omitempty"`
	// An optional system prompt the server wants to use for sampling. The client
	// may modify or omit this prompt.
	SystemPrompt string  `json:"systemPrompt,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
}

func (x *CreateMessageParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CreateMessageParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The client's response to a sampling/create_message request from the server.
// The client should inform the user before returning the sampled message, to
// allow them to inspect the response (human in the loop) and decide whether to
// allow the server to see it.
type CreateMessageResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta    `json:"_meta,omitempty"`
	Content Content `json:"content"`
	// The name of the model that generated the message.
	Model string `json:"model"`
	Role  Role   `json:"role"`
	// The reason why sampling stopped, if known.
	StopReason string `json:"stopReason,omitempty"`
}

type GetPromptParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Arguments to use for templating the prompt.
	Arguments map[string]string `json:"arguments,omitempty"`
	// The name of the prompt or prompt template.
	Name string `json:"name"`
}

func (x *GetPromptParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *GetPromptParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The server's response to a prompts/get request from the client.
type GetPromptResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An optional description for the prompt.
	Description string           `json:"description,omitempty"`
	Messages    []*PromptMessage `json:"messages"`
}

type InitializeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta         `json:"_meta,omitempty"`
	Capabilities *ClientCapabilities `json:"capabilities"`
	ClientInfo   *Implementation     `json:"clientInfo"`
	// The latest version of the Model Context Protocol that the client supports.
	// The client may decide to support older versions as well.
	ProtocolVersion string `json:"protocolVersion"`
}

func (x *InitializeParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *InitializeParams) SetProgressToken(t any) { setProgressToken(x, t) }

// After receiving an initialize request from the client, the server sends this
// response.
type InitializeResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta         `json:"_meta,omitempty"`
	Capabilities *serverCapabilities `json:"capabilities"`
	// Instructions describing how to use the server and its features.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// tools, resources, etc. It can be thought of like a "hint" to the model. For
	// example, this information may be added to the system prompt.
	Instructions string `json:"instructions,omitempty"`
	// The version of the Model Context Protocol that the server wants to use. This
	// may not match the version that the client requested. If the client cannot
	// support this version, it must disconnect.
	ProtocolVersion string          `json:"protocolVersion"`
	ServerInfo      *Implementation `json:"serverInfo"`
}

type InitializedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *InitializedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *InitializedParams) SetProgressToken(t any) { setProgressToken(x, t) }

type ListPromptsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListPromptsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListPromptsParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListPromptsParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a prompts/list request from the client.
type ListPromptsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string    `json:"nextCursor,omitempty"`
	Prompts    []*Prompt `json:"prompts"`
}

func (x *ListPromptsResult) nextCursorPtr() *string { return &x.NextCursor }

type ListResourceTemplatesParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListResourceTemplatesParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListResourceTemplatesParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListResourceTemplatesParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a resources/templates/list request from the client.
type ListResourceTemplatesResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor        string              `json:"nextCursor,omitempty"`
	ResourceTemplates []*ResourceTemplate `json:"resourceTemplates"`
}

func (x *ListResourceTemplatesResult) nextCursorPtr() *string { return &x.NextCursor }

type ListResourcesParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListResourcesParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListResourcesParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListResourcesParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a resources/list request from the client.
type ListResourcesResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string      `json:"nextCursor,omitempty"`
	Resources  []*Resource `json:"resources"`
}

func (x *ListResourcesResult) nextCursorPtr() *string { return &x.NextCursor }

type ListRootsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ListRootsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListRootsParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The client's response to a roots/list request from the server. This result
// contains an array of Root objects, each representing a root directory or file
// that the server can operate on.
type ListRootsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta  `json:"_meta,omitempty"`
	Roots []*Root `json:"roots"`
}

type ListToolsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListToolsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListToolsParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListToolsParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string  `json:"nextCursor,omitempty"`
	Tools      []*Tool `json:"tools"`
}

func (x *ListToolsResult) nextCursorPtr() *string { return &x.NextCursor }

// The severity of a log message.
//
// These map to syslog message severities, as specified in RFC-5424:
// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1
type LoggingLevel string

type LoggingMessageParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The data to be logged, such as a string message or an object. Any JSON
	// serializable type is allowed here.
	Data any `json:"data"`
	// The severity of this log message.
	Level LoggingLevel `json:"level"`
	// An optional name of the logger issuing this message.
	Logger string `json:"logger,omitempty"`
}

func (x *LoggingMessageParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *LoggingMessageParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Hints to use for model selection.
//
// Keys not declared here are currently left unspecified by the spec and are up
// to the client to interpret.
type ModelHint struct {
	// A hint for a model name.
	//
	// The client should treat this as a substring of a model name; for example: -
	// `claude-3-5-sonnet` should match `claude-3-5-sonnet-20241022` - `sonnet`
	// should match `claude-3-5-sonnet-20241022`, `claude-3-sonnet-20240229`, etc. -
	// `claude` should match any Claude model
	//
	// The client may also map the string to a different provider's model name or a
	// different model family, as long as it fills a similar niche; for example: -
	// `gemini-1.5-flash` could match `claude-3-haiku-20240307`
	Name string `json:"name,omitempty"`
}

// The server's preferences for model selection, requested of the client during
// sampling.
//
// Because LLMs can vary along multiple dimensions, choosing the "best" model is
// rarely straightforward. Different models excel in different areas—some are
// faster but less capable, others are more capable but more expensive, and so
// on. This interface allows servers to express their priorities across multiple
// dimensions to help clients make an appropriate selection for their use case.
//
// These preferences are always advisory. The client may ignore them. It is also
// up to the client to decide how to interpret these preferences and how to
// balance them against other considerations.
type ModelPreferences struct {
	// How much to prioritize cost when selecting a model. A value of 0 means cost
	// is not important, while a value of 1 means cost is the most important factor.
	CostPriority float64 `json:"costPriority,omitempty"`
	// Optional hints to use for model selection.
	//
	// If multiple hints are specified, the client must evaluate them in order (such
	// that the first match is taken).
	//
	// The client should prioritize these hints over the numeric priorities, but may
	// still use the priorities to select from ambiguous matches.
	Hints []*ModelHint `json:"hints,omitempty"`
	// How much to prioritize intelligence and capabilities when selecting a model.
	// A value of 0 means intelligence is not important, while a value of 1 means
	// intelligence is the most important factor.
	IntelligencePriority float64 `json:"intelligencePriority,omitempty"`
	// How much to prioritize sampling speed (latency) when selecting a model. A
	// value of 0 means speed is not important, while a value of 1 means speed is
	// the most important factor.
	SpeedPriority float64 `json:"speedPriority,omitempty"`
}

type PingParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *PingParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *PingParams) SetProgressToken(t any) { setProgressToken(x, t) }

type ProgressNotificationParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An optional message describing the current progress.
	Message string `json:"message,omitempty"`
	// The progress thus far. This should increase every time progress is made, even
	// if the total is unknown.
	Progress float64 `json:"progress"`
	// The progress token which was given in the initial request, used to associate
	// this notification with the request that is proceeding.
	ProgressToken any `json:"progressToken"`
	// Total number of items to process (or total progress required), if known.
	Total float64 `json:"total,omitempty"`
}

// A prompt or prompt template that the server offers.
type Prompt struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// A list of arguments to use for templating the prompt.
	Arguments []*PromptArgument `json:"arguments,omitempty"`
	// An optional description of what this prompt provides
	Description string `json:"description,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title string `json:"title,omitempty"`
}

// Describes an argument that a prompt can accept.
type PromptArgument struct {
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title string `json:"title,omitempty"`
	// A human-readable description of the argument.
	Description string `json:"description,omitempty"`
	// Whether this argument must be provided.
	Required bool `json:"required,omitempty"`
}

type PromptListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *PromptListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *PromptListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Describes a message returned as part of a prompt.
//
// This is similar to SamplingMessage, but also supports the embedding of
// resources from the MCP server.
type PromptMessage struct {
	Content Content `json:"content"`
	Role    Role    `json:"role"`
}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (m *PromptMessage) UnmarshalJSON(data []byte) error {
	type msg PromptMessage // avoid recursion
	var wire struct {
		msg
		Content *wireContent `json:"content"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.msg.Content, err = contentFromWire(wire.Content, nil); err != nil {
		return err
	}
	*m = PromptMessage(wire.msg)
	return nil
}

type ReadResourceParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to read. The URI can use any protocol; it is up to
	// the server how to interpret it.
	URI string `json:"uri"`
}

func (x *ReadResourceParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ReadResourceParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The server's response to a resources/read request from the client.
type ReadResourceResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta     `json:"_meta,omitempty"`
	Contents []*ResourceContents `json:"contents"`
}

// A known resource that the server is capable of reading.
type Resource struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitempty"`
	// A description of what this resource represents.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// The MIME type of this resource, if known.
	MIMEType string `json:"mimeType,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// The size of the raw resource content, in bytes (i.e., before base64 encoding
	// or any tokenization), if known.
	//
	// This can be used by Hosts to display file sizes and estimate context window
	// usage.
	Size int64 `json:"size,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool, where
	// Annotations.Title should be given precedence over using name, if
	// present).
	Title string `json:"title,omitempty"`
	// The URI of this resource.
	URI string `json:"uri"`
}

type ResourceListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ResourceListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ResourceListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// A template description for resources available on the server.
type ResourceTemplate struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitempty"`
	// A description of what this template is for.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// The MIME type for all resources that match this template. This should only be
	// included if all resources matching this template have the same type.
	MIMEType string `json:"mimeType,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool, where
	// Annotations.Title should be given precedence over using name, if
	// present).
	Title string `json:"title,omitempty"`
	// A URI template (according to RFC 6570) that can be used to construct resource
	// URIs.
	URITemplate string `json:"uriTemplate"`
}

// The sender or recipient of messages and data in a conversation.
type Role string

// Represents a root directory or file that the server can operate on.
type Root struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// An optional name for the root. This can be used to provide a human-readable
	// identifier for the root, which may be useful for display purposes or for
	// referencing the root in other parts of the application.
	Name string `json:"name,omitempty"`
	// The URI identifying the root. This *must* start with file:// for now. This
	// restriction may be relaxed in future versions of the protocol to allow other
	// URI schemes.
	URI string `json:"uri"`
}

type RootsListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *RootsListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *RootsListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// SamplingCapabilities describes the capabilities for sampling.
type SamplingCapabilities struct{}

// ElicitationCapabilities describes the capabilities for elicitation.
type ElicitationCapabilities struct{}

// Describes a message issued to or received from an LLM API.
type SamplingMessage struct {
	Content Content `json:"content"`
	Role    Role    `json:"role"`
}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (m *SamplingMessage) UnmarshalJSON(data []byte) error {
	type msg SamplingMessage // avoid recursion
	var wire struct {
		msg
		Content *wireContent `json:"content"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.msg.Content, err = contentFromWire(wire.Content, map[string]bool{"text": true, "image": true, "audio": true}); err != nil {
		return err
	}
	*m = SamplingMessage(wire.msg)
	return nil
}

type SetLevelParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The level of logging that the client wants to receive from the server. The
	// server should send all logs at this level and higher (i.e., more severe) to
	// the client as notifications/message.
	Level LoggingLevel `json:"level"`
}

func (x *SetLevelParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *SetLevelParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Definition for a tool the client can call.
type Tool struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional additional tool information.
	//
	// Display name precedence order is: title, annotations.title, then name.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
	// A human-readable description of the tool.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// tools. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema *jsonschema.Schema `json:"inputSchema"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// An optional JSON Schema object defining the structure of the tool's output
	// returned in the structuredContent field of a CallToolResult.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	// If not provided, Annotations.Title should be used for display if present,
	// otherwise Name.
	Title string `json:"title,omitempty"`
}

// Additional properties describing a Tool to clients.
//
// NOTE: all properties in ToolAnnotations are hints. They are not
// guaranteed to provide a faithful description of tool behavior (including
// descriptive properties like title).
//
// Clients should never make tool use decisions based on ToolAnnotations
// received from untrusted servers.
type ToolAnnotations struct {
	// If true, the tool may perform destructive updates to its environment. If
	// false, the tool performs only additive updates.
	//
	// (This property is meaningful only when ReadOnlyHint == false.)
	//
	// Default: true
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// If true, calling the tool repeatedly with the same arguments will have no
	// additional effect on the its environment.
	//
	// (This property is meaningful only when ReadOnlyHint == false.)
	//
	// Default: false
	IdempotentHint bool `json:"idempotentHint,omitempty"`
	// If true, this tool may interact with an "open world" of external entities. If
	// false, the tool's domain of interaction is closed. For example, the world of
	// a web search tool is open, whereas that of a memory tool is not.
	//
	// Default: true
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
	// If true, the tool does not modify its environment.
	//
	// Default: false
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
	// A human-readable title for the tool.
	Title string `json:"title,omitempty"`
}

type ToolListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ToolListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ToolListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Sent from the client to request resources/updated notifications from the
// server whenever a particular resource changes.
type SubscribeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to subscribe to.
	URI string `json:"uri"`
}

// Sent from the client to request cancellation of resources/updated
// notifications from the server. This should follow a previous
// resources/subscribe request.
type UnsubscribeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to unsubscribe from.
	URI string `json:"uri"`
}

// A notification from the server to the client, informing it that a resource
// has changed and may need to be read again. This should only be sent if the
// client previously sent a resources/subscribe request.
type ResourceUpdatedNotificationParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource that has been updated. This might be a sub-resource of the one that the client actually subscribed to.
	URI string `json:"uri"`
}

// TODO(jba): add CompleteRequest and related types.

// TODO(jba): add ElicitRequest and related types.

// An Implementation describes the name and version of an MCP implementation, with an optional
// title for UI representation.
type Implementation struct {
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// Present if the server supports argument autocompletion suggestions.
type completionCapabilities struct{}

// Present if the server supports sending log messages to the client.
type loggingCapabilities struct{}

// Present if the server offers any prompt templates.
type promptCapabilities struct {
	// Whether this server supports notifications for changes to the prompt list.
	ListChanged bool `json:"listChanged,omitempty"`
}

// Present if the server offers any resources to read.
type resourceCapabilities struct {
	// Whether this server supports notifications for changes to the resource list.
	ListChanged bool `json:"listChanged,omitempty"`
	// Whether this server supports subscribing to resource updates.
	Subscribe bool `json:"subscribe,omitempty"`
}

// Capabilities that a server may support. Known capabilities are defined here,
// in this schema, but this is not a closed set: any server can define its own,
// additional capabilities.
type serverCapabilities struct {
	// Present if the server supports argument autocompletion suggestions.
	Completions *completionCapabilities `json:"completions,omitempty"`
	// Experimental, non-standard capabilities that the server supports.
	Experimental map[string]struct{} `json:"experimental,omitempty"`
	// Present if the server supports sending log messages to the client.
	Logging *loggingCapabilities `json:"logging,omitempty"`
	// Present if the server offers any prompt templates.
	Prompts *promptCapabilities `json:"prompts,omitempty"`
	// Present if the server offers any resources to read.
	Resources *resourceCapabilities `json:"resources,omitempty"`
	// Present if the server offers any tools to call.
	Tools *toolCapabilities `json:"tools,omitempty"`
}

// Present if the server offers any tools to call.
type toolCapabilities struct {
	// Whether this server supports notifications for changes to the tool list.
	ListChanged bool `json:"listChanged,omitempty"`
}

const (
	methodCallTool                  = "tools/call"
	notificationCancelled           = "notifications/cancelled"
	methodComplete                  = "completion/complete"
	methodCreateMessage             = "sampling/createMessage"
	methodElicit                    = "elicitation/create"
	methodGetPrompt                 = "prompts/get"
	methodInitialize                = "initialize"
	notificationInitialized         = "notifications/initialized"
	methodListPrompts               = "prompts/list"
	methodListResourceTemplates     = "resources/templates/list"
	methodListResources             = "resources/list"
	methodListRoots                 = "roots/list"
	methodListTools                 = "tools/list"
	notificationLoggingMessage      = "notifications/message"
	methodPing                      = "ping"
	notificationProgress            = "notifications/progress"
	notificationPromptListChanged   = "notifications/prompts/list_changed"
	methodReadResource              = "resources/read"
	notificationResourceListChanged = "notifications/resources/list_changed"
	notificationResourceUpdated     = "notifications/resources/updated"
	notificationRootsListChanged    = "notifications/roots/list_changed"
	methodSetLevel                  = "logging/setLevel"
	methodSubscribe                 = "resources/subscribe"
	notificationToolListChanged     = "notifications/tools/list_changed"
	methodUnsubscribe               = "resources/unsubscribe"
)
