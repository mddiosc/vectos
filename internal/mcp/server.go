package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// JSONRPCRequest representa una petición estándar del protocolo MCP.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse representa una respuesta estándar del protocolo MCP.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError define el formato de error del protocolo.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool representa una herramienta que el agente puede llamar.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ListToolsResponse es la respuesta para el método tools/list.
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams es el formato de parámetros para tools/call.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// InitializeResult es la respuesta para initialize.
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]string      `json:"serverInfo"`
}

// InitializeParams representa los parámetros del handshake initialize.
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

// ToolContent representa contenido textual devuelto por una herramienta MCP.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolCallResult representa la respuesta estándar de tools/call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolHandler es la firma de la función que ejecuta una herramienta.
type ToolHandler func(args map[string]interface{}) (interface{}, error)

// Server es el núcleo del servidor MCP que gestiona la comunicación.
type Server struct {
	reader   *bufio.Reader
	writer   io.Writer
	tools    map[string]Tool
	handlers map[string]ToolHandler
	version  string
}

// NewServer crea una nueva instancia del servidor MCP.
// version es el string de versión del servidor (e.g. "v0.1.0").
func NewServer(input io.Reader, output io.Writer, version string) *Server {
	v := version
	if v == "" {
		v = "dev"
	}
	return &Server{
		reader:   bufio.NewReader(input),
		writer:   output,
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
		version:  v,
	}
}

// RegisterTool permite añadir herramientas al servidor.
func (s *Server) RegisterTool(name string, description string, schema map[string]interface{}, handler ToolHandler) {
	s.tools[name] = Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}
	s.handlers[name] = handler
}

// Listen inicia el bucle de escucha de peticiones MCP sobre stdio con framing Content-Length.
func (s *Server) Listen() error {
	for {
		payload, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return fmt.Errorf("failed to decode request: %w", err)
		}

		log.Printf("mcp request method=%s id=%v", req.Method, req.ID)

		resp, send, err := s.handleRequest(req)
		if err != nil {
			resp = JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32603, Message: err.Error()},
			}
			send = req.ID != nil
		}

		if send {
			log.Printf("mcp response method=%s id=%v", req.Method, req.ID)
			if err := s.writeMessage(resp); err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
		}
	}
}

func (s *Server) readMessage() ([]byte, error) {
	first, err := s.peekNonWhitespaceByte()
	if err != nil {
		return nil, err
	}

	if first == '{' {
		return s.readJSONLine()
	}

	return s.readContentLengthMessage()
}

func (s *Server) readJSONLine() ([]byte, error) {
	line, err := s.reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}

func (s *Server) readContentLengthMessage() ([]byte, error) {
	contentLength, err := s.readContentLengthHeaders()
	if err != nil {
		return nil, err
	}
	return s.readExactPayload(contentLength)
}

func (s *Server) readContentLengthHeaders() (int, error) {
	contentLength := 0
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		parsed, err := s.parseHeaderContentLength(trimmed)
		if err != nil {
			return 0, err
		}
		if parsed > 0 {
			contentLength = parsed
		}
	}

	if contentLength <= 0 {
		return 0, fmt.Errorf("missing content-length header")
	}
	return contentLength, nil
}

func (s *Server) parseHeaderContentLength(line string) (int, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid header line: %s", line)
	}

	name := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])
	if name != "content-length" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid content-length: %w", err)
	}
	return parsed, nil
}

func (s *Server) readExactPayload(contentLength int) ([]byte, error) {
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Server) peekNonWhitespaceByte() (byte, error) {
	for {
		b, err := s.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if err := s.reader.UnreadByte(); err != nil {
			return 0, err
		}
		return b, nil
	}
}

func (s *Server) writeMessage(v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	if _, err := buf.Write(payload); err != nil {
		return err
	}

	_, err = s.writer.Write(buf.Bytes())
	return err
}

func (s *Server) handleRequest(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return s.handleNotInitialized(req)
	case "ping":
		return s.handlePing(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return JSONRPCResponse{}, true, fmt.Errorf("method not found: %s", req.Method)
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return JSONRPCResponse{}, true, fmt.Errorf("invalid initialize parameters: %w", err)
		}
	}
	log.Printf("mcp initialize params protocolVersion=%s clientInfo=%v", params.ProtocolVersion, params.ClientInfo)

	protocolVersion := params.ProtocolVersion
	if protocolVersion == "" {
		protocolVersion = "2024-11-05"
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			ServerInfo: map[string]string{
				"name":    "vectos",
				"version": s.version,
			},
		},
	}, true, nil
}

func (s *Server) handleNotInitialized(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	return JSONRPCResponse{}, false, nil
}

func (s *Server) handlePing(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}, true, nil
}

func (s *Server) handleToolsList(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	var toolsList []Tool
	for _, tool := range s.tools {
		toolsList = append(toolsList, tool)
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListToolsResponse{Tools: toolsList},
	}, true, nil
}

func (s *Server) handleToolsCall(req JSONRPCRequest) (JSONRPCResponse, bool, error) {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCResponse{}, true, fmt.Errorf("invalid tools/call parameters: %w", err)
	}

	handler, ok := s.handlers[params.Name]
	if !ok {
		return JSONRPCResponse{}, true, fmt.Errorf("tool not found: %s", params.Name)
	}

	result, err := handler(params.Arguments)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}, true, nil
	}

	text, err := stringifyResult(result)
	if err != nil {
		return JSONRPCResponse{}, true, fmt.Errorf("failed to encode tool result: %w", err)
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: text}},
		},
	}, true, nil
}

func stringifyResult(result interface{}) (string, error) {
	if text, ok := result.(string); ok {
		return text, nil
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
