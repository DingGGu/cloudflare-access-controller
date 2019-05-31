package option_test

import (
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestAccessAnnotationGetDomainStripSlash(t *testing.T) {
	ann := option.AccessAnnotation{
		ZoneName:             "ggu.la",
		ApplicationSubDomain: "love",
		ApplicationPath:      "/untz",
	}

	assert.Equal(t, ann.GetDomain(), "love.ggu.la/untz", "Expected GetDomain() 'love.ggu.la/untz', Got %s", ann.GetDomain())
}

func TestAccessAnnotationGetDomain(t *testing.T) {
	ann := option.AccessAnnotation{
		ZoneName:             "ggu.la",
		ApplicationSubDomain: "love",
		ApplicationPath:      "untz",
	}

	assert.Equal(t, ann.GetDomain(), "love.ggu.la/untz", "Expected GetDomain() 'love.ggu.la/untz', Got %s", ann.GetDomain())
}
