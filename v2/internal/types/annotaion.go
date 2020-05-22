package types

var (
	AnnotationPrefix                  = "access.cloudflare.com/"
	AnnotationApplicationSubDomain    = AnnotationPrefix + "application-sub-domain"
	AnnotationApplicationPath         = AnnotationPrefix + "application-path"
	AnnotationSessionDuration         = AnnotationPrefix + "session-duration"
	AnnotationSessionPolicies         = AnnotationPrefix + "policies"
	AnnotationSessionAllowEmailDomain = AnnotationPrefix + "allow-email-domain"
)
