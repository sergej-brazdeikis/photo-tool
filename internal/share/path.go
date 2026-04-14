package share

import "strconv"

// ShareHTTPPath returns the URL path for a loopback share link (Story 3.2: GET http://127.0.0.1:{port}/s/{token}).
// Story 3.1 keeps mint/copy token-only; callers that build a full URL join base + this path in 3.2.
// Minted tokens use base64 raw URL encoding (A–Z, a–z, 0–9, -, _), so the segment needs no extra escaping for a single path component.
func ShareHTTPPath(rawURLSafeToken string) string {
	return "/s/" + rawURLSafeToken
}

// ShareImageHTTPPath returns the same-origin path for share image bytes (Story 3.3: GET /i/{token}).
func ShareImageHTTPPath(rawURLSafeToken string) string {
	return "/i/" + rawURLSafeToken
}

// SharePackageMemberImagePath returns GET /i/{token}/{position} for package member bytes (Story 4.1).
func SharePackageMemberImagePath(rawURLSafeToken string, position int) string {
	if position < 0 {
		position = 0
	}
	return "/i/" + rawURLSafeToken + "/" + strconv.Itoa(position)
}
