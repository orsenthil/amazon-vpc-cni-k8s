package awssession

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

func TestHttpTimeoutReturnDefault(t *testing.T) {
	err := os.Setenv(httpTimeoutEnv, "2")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv(httpTimeoutEnv)
		if err != nil {
			t.Error(err)
		}
	}()
	expectedHTTPTimeOut := time.Duration(10) * time.Second
	assert.Equal(t, expectedHTTPTimeOut, getHTTPTimeout())
}

func TestHttpTimeoutWithValueAbove10(t *testing.T) {
	err := os.Setenv(httpTimeoutEnv, "12")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv(httpTimeoutEnv)
		if err != nil {
			t.Error(err)
		}
	}()
	expectedHTTPTimeOut := time.Duration(12) * time.Second
	assert.Equal(t, expectedHTTPTimeOut, getHTTPTimeout())
}

func TestAwsEc2EndpointResolver(t *testing.T) {
	customEndpoint := "https://ec2.us-west-2.customaws.com"

	err := os.Setenv("AWS_EC2_ENDPOINT", customEndpoint)
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv("AWS_EC2_ENDPOINT")
		if err != nil {
			t.Error(err)
		}
	}()

	sess := NewV1()

	resolvedEndpoint, err := sess.Config.EndpointResolver.EndpointFor(ec2.EndpointsID, "")
	assert.NoError(t, err)
	assert.Equal(t, customEndpoint, resolvedEndpoint.URL)

	ctx := context.Background()
	config, err := New(ctx)
	assert.NoError(t, err)
	endpoint, err := config.EndpointResolver.ResolveEndpoint(ec2.ServiceID, "")
	assert.NoError(t, err)
	assert.Equal(t, customEndpoint, endpoint.URL)
}

func TestCustomEndpointResolver(t *testing.T) {
	customEndpoint := "https://ec2.us-west-2.customaws.com"
	_ = os.Setenv("AWS_EC2_ENDPOINT", customEndpoint)
	defer func() {
		err := os.Unsetenv("AWS_EC2_ENDPOINT")
		if err != nil {
			t.Error(err)
		}
	}()

	ctx := context.Background()
	config, err := New(ctx)
	assert.NoError(t, err)

	endpoint, err := config.EndpointResolver.ResolveEndpoint(ec2.ServiceID, "")
	assert.NoError(t, err)
	assert.Equal(t, customEndpoint, endpoint.URL)
}
