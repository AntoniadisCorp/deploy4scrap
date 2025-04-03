package domain

// Define a custom response structure
// Standardized HTTP response for APIs in large-scale applications
type HTTPResponse struct {
	// Headers map[string]string `json:"headers"`
	Code          int         `json:"code"`                    // HTTP status code
	Status        string      `json:"status"`                  // 'success' or 'error'
	Message       string      `json:"message,omitempty"`       // Optional message for clients
	Data          interface{} `json:"data,omitempty"`          // Response data for successful operations
	Errors        APIError    `json:"errors,omitempty"`        // List of errors (if any)
	Meta          *MetaData   `json:"meta,omitempty"`          // Optional metadata (pagination, etc.)
	TraceID       string      `json:"traceId,omitempty"`       // For distributed tracing in microservices
	View          string      `json:"view,omitempty"`          // Optional view name for rendering
	DomainHost    string      `json:"domainHost,omitempty"`    // Optional view name for rendering
	SubDomainHost string      `json:"subDomainHost,omitempty"` // Optional view name for rendering
}

// APIError defines the structure for returning errors in the response
type APIError struct {
	Code    string `json:"code"`    // Error code for internal reference
	Message string `json:"message"` // User-friendly error message
	Detail  string `json:"detail"`  // Optional detailed error message (for developers)
}

// MetaData defines optional metadata for responses, like pagination
type MetaData struct {
	TotalItems int `json:"totalItems,omitempty"` // Total number of items (for paginated responses)
	Page       int `json:"page,omitempty"`       // Current page
	PageSize   int `json:"pageSize,omitempty"`   // Number of items per page
}
