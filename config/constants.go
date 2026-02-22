package config

import "time"

// Configuration constants for the application
const (
	// LLM Configuration
	MaxToolRounds = 100 // Maximum number of tool execution rounds per LLM response

	// File Reading Configuration
	MaxDefaultLines = 2000 // Default maximum lines to read when no limit is specified

	// UI Configuration
	TextareaHeight  = 3  // Height of the input textarea
	MaxResultLines  = 10 // Maximum lines to display for tool results before truncating
	MinBoxWidth     = 30 // Minimum width for UI boxes
	BoxPadding      = 4  // Padding for UI boxes (2 sides)

	// Tool Icons
	ToolIcon   = "🔧 "
	EditIcon   = "✏️ "
	WriteIcon  = "📝 "
	ReadIcon   = "📖 "
	ListIcon   = "📁 "
	BashIcon   = "💻 "
	SearchIcon = "🔍 "

	// API Configuration
	MaxContextTokens = 128000

	// Retry Configuration
	MaxRetryAttempts  = 3                    // Maximum number of retry attempts
	RetryBaseDelay    = 500 * time.Millisecond // Initial delay before first retry
	RetryMaxDelay     = 10 * time.Second      // Maximum delay between retries
	RetryJitter       = 0.1                   // Jitter factor to add randomness to delays

	// Circuit Breaker Configuration
	CircuitMaxFailures     = 5                   // Number of consecutive failures to open the circuit
	CircuitSuccessThreshold = 2                  // Number of successes needed in half-open to close the circuit
	CircuitOpenTimeout     = 30 * time.Second    // Duration to wait before transitioning from OPEN to HALF_OPEN

	// File Permissions
	DirPermissions  = 0o755 // Directory permissions
	FilePermissions = 0o644 // File permissions
	LogPermissions  = 0o644 // Log file permissions
)
