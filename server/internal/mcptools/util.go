package mcptools

import (
	"fmt"

	"github.com/eberle1080/jsonrpc"
	"github.com/eberle1080/mcp-protocol/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func textResult(format string, args ...interface{}) *schema.CallToolResult {
	return &schema.CallToolResult{
		Content: []schema.CallToolResultContentElem{
			{Type: "text", Text: fmt.Sprintf(format, args...)},
		},
	}
}

func errResult(format string, args ...interface{}) *schema.CallToolResult {
	isErr := true
	return &schema.CallToolResult{
		IsError: &isErr,
		Content: []schema.CallToolResultContentElem{
			{Type: "text", Text: fmt.Sprintf(format, args...)},
		},
	}
}

func jsonrpcError(err error) *jsonrpc.Error {
	st, ok := status.FromError(err)
	if !ok {
		return jsonrpc.NewInternalError(err.Error(), nil)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return jsonrpc.NewInvalidParamsError(st.Message(), nil)
	case codes.PermissionDenied:
		return jsonrpc.NewError(-32003, st.Message(), nil)
	default:
		return jsonrpc.NewInternalError(st.Message(), nil)
	}
}
