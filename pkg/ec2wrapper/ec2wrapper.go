// Package ec2wrapper is used to wrap around the ec2 service APIs
package ec2wrapper

import (
	"context"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils/awssession"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/ec2metadatawrapper"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/utils/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	v2ec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	ec2metadatav1 "github.com/aws/aws-sdk-go/aws/ec2metadata"
	ec2v1 "github.com/aws/aws-sdk-go/service/ec2"
	ec2ifacev1 "github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"

	v2ec2metadata "github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

const (
	resourceID   = "resource-id"
	resourceKey  = "key"
	clusterIDTag = "CLUSTER_ID"
)

var log = logger.Get()

// EC2WrapperV1 is used to wrap around EC2 service APIs to obtain ClusterID from
// the ec2 instance tags
type EC2WrapperV1 struct {
	ec2ServiceClient         ec2ifacev1.EC2API
	instanceIdentityDocument ec2metadatav1.EC2InstanceIdentityDocument
}

// NewMetricsClientV1 returns an instance of the EC2 wrapper
func NewMetricsClientV1() (*EC2WrapperV1, error) {
	sess := awssession.NewV1()
	ec2MetadataClient := ec2metadatawrapper.NewV1(sess)

	instanceIdentityDocument, err := ec2MetadataClient.GetInstanceIdentityDocument()
	if err != nil {
		return &EC2WrapperV1{}, err
	}

	awsCfg := awsv1.NewConfig().WithRegion(instanceIdentityDocument.Region)
	sess = sess.Copy(awsCfg)
	ec2ServiceClient := ec2v1.New(sess)

	return &EC2WrapperV1{
		ec2ServiceClient:         ec2ServiceClient,
		instanceIdentityDocument: instanceIdentityDocument,
	}, nil
}

// GetClusterTagV1 is used to retrieve a tag from the ec2 instance
func (e *EC2WrapperV1) GetClusterTagV1(tagKey string) (string, error) {
	input := ec2v1.DescribeTagsInput{
		Filters: []*ec2v1.Filter{
			{
				Name: awsv1.String(resourceID),
				Values: []*string{
					awsv1.String(e.instanceIdentityDocument.InstanceID),
				},
			}, {
				Name: awsv1.String(resourceKey),
				Values: []*string{
					awsv1.String(tagKey),
				},
			},
		},
	}

	log.Infof("Calling DescribeTags with key %s", tagKey)
	results, err := e.ec2ServiceClient.DescribeTags(&input)
	if err != nil {
		return "", errors.Wrap(err, "GetClusterTagV1: Unable to obtain EC2 instance tags")
	}

	if len(results.Tags) < 1 {
		return "", errors.Errorf("GetClusterTagV1: No tag matching key: %s", tagKey)
	}

	return awsv1.StringValue(results.Tags[0].Value), nil
}

// EC2Wrapper is used to wrap around EC2 service APIs to obtain ClusterID from
// the ec2 instance tags
type EC2Wrapper struct {
	ec2ServiceClient         *v2ec2.Client
	instanceIdentityDocument *v2ec2metadata.GetInstanceIdentityDocumentOutput
}

// NewMetricsClient returns an instance of the EC2 wrapper
func NewMetricsClient() (*EC2Wrapper, error) {
	ctx := context.TODO()
	cfg, err := awssession.New(ctx)
	if err != nil {
		return nil, err
	}

	ec2MetadataClient, err := ec2metadatawrapper.New(ctx)

	instanceIdentityDocument, err := ec2MetadataClient.GetInstanceIdentityDocument(context.TODO())

	if err != nil {
		return &EC2Wrapper{}, err
	}

	cfg.Region = instanceIdentityDocument.Region
	ec2ServiceClient := v2ec2.NewFromConfig(cfg)

	return &EC2Wrapper{
		ec2ServiceClient:         ec2ServiceClient,
		instanceIdentityDocument: instanceIdentityDocument,
	}, nil
}

// GetClusterTag is used to retrieve a tag from the ec2 instance
func (e *EC2Wrapper) GetClusterTag(tagKey string) (string, error) {
	input := &v2ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name: aws.String(resourceID),
				Values: []string{
					e.instanceIdentityDocument.InstanceID,
				},
			},
			{
				Name: aws.String(resourceKey),
				Values: []string{
					tagKey,
				},
			},
		},
	}

	log.Infof("Calling DescribeTags with key %s", tagKey)
	results, err := e.ec2ServiceClient.DescribeTags(context.TODO(), input)
	if err != nil {
		return "", errors.Wrap(err, "GetClusterTag: Unable to obtain EC2 instance tags")
	}

	if len(results.Tags) < 1 {
		return "", errors.Errorf("GetClusterTag: No tag matching key: %s", tagKey)
	}

	return aws.ToString(results.Tags[0].Value), nil
}
