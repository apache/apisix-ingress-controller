package _const

const (
	// OpEqual means the equal ("==") operator in nginxVars.
	OpEqual = "Equal"
	// OpNotEqual means the not equal ("~=") operator in nginxVars.
	OpNotEqual = "NotEqual"
	// OpGreaterThan means the greater than (">") operator in nginxVars.
	OpGreaterThan = "GreaterThan"
	// OpGreaterThanEqual means the greater than (">=") operator in nginxVars.
	OpGreaterThanEqual = "GreaterThanEqual"
	// OpLessThan means the less than ("<") operator in nginxVars.
	OpLessThan = "LessThan"
	// OpLessThanEqual means the less than equal ("<=") operator in nginxVars.
	OpLessThanEqual = "LessThanEqual"
	// OpRegexMatch means the regex match ("~~") operator in nginxVars.
	OpRegexMatch = "RegexMatch"
	// OpRegexNotMatch means the regex not match ("!~~") operator in nginxVars.
	OpRegexNotMatch = "RegexNotMatch"
	// OpRegexMatchCaseInsensitive means the regex match "~*" (case insensitive mode) operator in nginxVars.
	OpRegexMatchCaseInsensitive = "RegexMatchCaseInsensitive"
	// OpRegexNotMatchCaseInsensitive means the regex not match "!~*" (case insensitive mode) operator in nginxVars.
	OpRegexNotMatchCaseInsensitive = "RegexNotMatchCaseInsensitive"
	// OpIn means the in operator ("in") in nginxVars.
	OpIn = "In"
	// OpNotIn means the not in operator ("not_in") in nginxVars.
	OpNotIn = "NotIn"

	// ScopeQuery means the route match expression subject is in the querystring.
	ScopeQuery = "Query"
	// ScopeHeader means the route match expression subject is in request headers.
	ScopeHeader = "Header"
	// ScopePath means the route match expression subject is the uri path.
	ScopePath = "Path"
	// ScopeCookie means the route match expression subject is in cookie.
	ScopeCookie = "Cookie"
)
