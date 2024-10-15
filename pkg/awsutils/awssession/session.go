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
	"fmt"
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
	injectUserAgent(&sess.Handlers)

	return sess
}

// injectUserAgent will inject app specific user-agent into awsSDK
func injectUserAgent(handlers *requestv1.Handlers) {
	version := utils.GetEnv(envVpcCniVersion, "")
	handlers.Build.PushFrontNamed(requestv1.NamedHandler{
		Name: fmt.Sprintf("%s/user-agent", "amazon-vpc-cni-k8s"),
		Fn: requestv1.MakeAddToUserAgentHandler(
			"amazon-vpc-cni-k8s",
			"version/"+version),
	})
}
