// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package awssession

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"net/http"
	"os"

	"strconv"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/utils/logger"
	"github.com/aws/amazon-vpc-cni-k8s/utils"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	endpointsv1 "github.com/aws/aws-sdk-go/aws/endpoints"
	requestv1 "github.com/aws/aws-sdk-go/aws/request"
	sessionv1 "github.com/aws/aws-sdk-go/aws/session"
	ec2v1 "github.com/aws/aws-sdk-go/service/ec2"
)

// Http client timeout env for sessions
const (
	httpTimeoutEnv   = "HTTP_TIMEOUT"
	maxRetries       = 10
	envVpcCniVersion = "VPC_CNI_VERSION"
)

var (
	log = logger.Get()
	// HTTP timeout default value in seconds (10 seconds)
	httpTimeoutValue = 10 * time.Second
)

func getHTTPTimeout() time.Duration {
	httpTimeoutEnvInput := os.Getenv(httpTimeoutEnv)
	// if httpTimeout is not empty, we convert value to int and overwrite default httpTimeoutValue
	if httpTimeoutEnvInput != "" {
		input, err := strconv.Atoi(httpTimeoutEnvInput)
		if err == nil && input >= 10 {
			log.Debugf("Using HTTP_TIMEOUT %v", input)
			httpTimeoutValue = time.Duration(input) * time.Second
			return httpTimeoutValue
		}
	}
	log.Warn("HTTP_TIMEOUT env is not set or set to less than 10 seconds, defaulting to httpTimeout to 10sec")
	return httpTimeoutValue
}

func New(ctx context.Context) (aws.Config, error) {
	customHTTPClient := &http.Client{
		Timeout: getHTTPTimeout()}
	optFns := []func(*config.LoadOptions) error{
		config.WithHTTPClient(customHTTPClient),
		config.WithRetryMaxAttempts(maxRetries),
		config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard()
		}),
		injectUserAgent,
	}

	endpoint := os.Getenv("AWS_EC2_ENDPOINT")

	//TODO (senthilx)  - The endpoint resolver is using deprecated method, this should be moved to the services.
	if endpoint != "" {
		optFns = append(optFns, config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				if service == ec2.ServiceID {
					return aws.Endpoint{
						URL: endpoint,
					}, nil
				}
				// Fall back to default resolution
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			})))

	}

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)

	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}

// NewV1 will return an session for service clients
func NewV1() *sessionv1.Session {
	awsCfg := awsv1.Config{
		MaxRetries: awsv1.Int(maxRetries),
		HTTPClient: &http.Client{
			Timeout: getHTTPTimeout(),
		},
		STSRegionalEndpoint: endpointsv1.RegionalSTSEndpoint,
	}

	endpoint := os.Getenv("AWS_EC2_ENDPOINT")
	if endpoint != "" {
		customResolver := func(service, region string, optFns ...func(*endpointsv1.Options)) (endpointsv1.ResolvedEndpoint, error) {
			if service == ec2v1.EndpointsID {
				return endpointsv1.ResolvedEndpoint{
					URL: endpoint,
				}, nil
			}
			return endpointsv1.DefaultResolver().EndpointFor(service, region, optFns...)
		}
		awsCfg.EndpointResolver = endpointsv1.ResolverFunc(customResolver)
	}

	sess := sessionv1.Must(sessionv1.NewSession(&awsCfg))
	//injecting session handler info
	injectUserAgentV1(&sess.Handlers)

	return sess
}

// injectUserAgent will inject app specific user-agent into awsSDK
func injectUserAgentV1(handlers *requestv1.Handlers) {
	version := utils.GetEnv(envVpcCniVersion, "")
	handlers.Build.PushFrontNamed(requestv1.NamedHandler{
		Name: fmt.Sprintf("%s/user-agent", "amazon-vpc-cni-k8s"),
		Fn: requestv1.MakeAddToUserAgentHandler(
			"amazon-vpc-cni-k8s",
			"version/"+version),
	})
}

// injectUserAgent will inject app specific user-agent into awsSDK
func injectUserAgent(loadOptions *config.LoadOptions) error {
	version := utils.GetEnv(envVpcCniVersion, "")
	userAgent := fmt.Sprintf("amazon-vpc-cni-k8s/version/%s", version)

	loadOptions.APIOptions = append(loadOptions.APIOptions, func(stack *smithymiddleware.Stack) error {
		return stack.Build.Add(&addUserAgentMiddleware{
			userAgent: userAgent,
		}, smithymiddleware.After)
	})

	return nil
}

type addUserAgentMiddleware struct {
	userAgent string
}

func (m *addUserAgentMiddleware) HandleBuild(ctx context.Context, in smithymiddleware.BuildInput, next smithymiddleware.BuildHandler) (out smithymiddleware.BuildOutput, metadata smithymiddleware.Metadata, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *addUserAgentMiddleware) ID() string {
	return "AddUserAgent"
}

func (m *addUserAgentMiddleware) HandleFinalize(ctx context.Context, in smithymiddleware.FinalizeInput, next smithymiddleware.FinalizeHandler) (
	out smithymiddleware.FinalizeOutput, metadata smithymiddleware.Metadata, err error) {
	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return out, metadata, &smithy.SerializationError{Err: fmt.Errorf("unknown request type %T", in.Request)}
	}

	userAgent := req.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = m.userAgent
	} else {
		userAgent += " " + m.userAgent
	}
	req.Header.Set("User-Agent", userAgent)

	return next.HandleFinalize(ctx, in)
}
