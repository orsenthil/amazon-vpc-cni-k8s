// Package ec2wrapper is used to wrap around the ec2 service APIs
package ec2wrapper

import (
	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils/awssession"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/ec2metadatawrapper"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/utils/logger"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	ec2metadatav1 "github.com/aws/aws-sdk-go/aws/ec2metadata"
	ec2v1 "github.com/aws/aws-sdk-go/service/ec2"
	ec2ifacev1 "github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
)

const (
	resourceID   = "resource-id"
	resourceKey  = "key"
	clusterIDTag = "CLUSTER_ID"
)

var log = logger.Get()

// EC2Wrapper is used to wrap around EC2 service APIs to obtain ClusterID from
// the ec2 instance tags
type EC2Wrapper struct {
	ec2ServiceClient         ec2ifacev1.EC2API
	instanceIdentityDocument ec2metadatav1.EC2InstanceIdentityDocument
}

// NewMetricsClientV1 returns an instance of the EC2 wrapper
func NewMetricsClientV1() (*EC2Wrapper, error) {
	sess := awssession.NewV1()
	ec2MetadataClient := ec2metadatawrapper.NewV1(sess)

	instanceIdentityDocument, err := ec2MetadataClient.GetInstanceIdentityDocument()
	if err != nil {
		return &EC2Wrapper{}, err
	}

	awsCfg := awsv1.NewConfig().WithRegion(instanceIdentityDocument.Region)
	sess = sess.Copy(awsCfg)
	ec2ServiceClient := ec2v1.New(sess)

	return &EC2Wrapper{
		ec2ServiceClient:         ec2ServiceClient,
		instanceIdentityDocument: instanceIdentityDocument,
	}, nil
}

// GetClusterTagV1 is used to retrieve a tag from the ec2 instance
func (e *EC2Wrapper) GetClusterTagV1(tagKey string) (string, error) {
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
