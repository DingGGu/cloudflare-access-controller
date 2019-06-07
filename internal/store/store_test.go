package store_test

import (
	"encoding/json"
	"github.com/DingGGu/cloudflare-access-controller/internal/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEqual(t *testing.T) {
	assert.Equal(t, store.Equal(nil, make([]interface{}, 0)), true)
	assert.Equal(t, store.Equal(nil, nil), true)
	assert.Equal(t, store.Equal(make([]interface{}, 0), make([]interface{}, 0)), true)

	var a1, b1, a2, b2, a3, b3 []interface{}
	var err error
	err = json.Unmarshal([]byte("[]"), &a1)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("{}"), &b1)
	assert.Error(t, err)
	assert.Equal(t, store.Equal(a1, b1), true)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"a.hye\"}}]"), &a2)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}}]"), &b2)
	assert.Empty(t, err)
	assert.Equal(t, store.Equal(a2, b2), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.122\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, store.Equal(a3, b3), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.122\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, store.Equal(a3, b3), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, store.Equal(a3, b3), true)
}
