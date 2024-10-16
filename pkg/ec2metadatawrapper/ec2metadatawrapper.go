// Package ec2metadatawrapper is used to retrieve data from EC2 IMDS
package ec2metadatawrapper

import (
	v2config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	"context"

	v2ec2imds "github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

// TODO: Move away from using mock

// HTTPClientV1 is used to help with testing
type HTTPClientV1 interface {
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
	Region() (string, error)
}

// EC2MetadataClientV1 to used to obtain a subset of information from EC2 IMDS
type EC2MetadataClientV1 interface {
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
	Region() (string, error)
}

type ec2MetadataClientImplV1 struct {
	client HTTPClientV1
}

// NewV1 creates an ec2metadata client to retrieve metadata
func NewV1(session *session.Session) EC2MetadataClientV1 {
	metadata := ec2metadata.New(session)
	return NewMetadataServiceV1(metadata)
}

// NewMetadataServiceV1 creates an ec2metadata client to retrieve metadata
func NewMetadataServiceV1(metadata HTTPClientV1) EC2MetadataClientV1 {
	return &ec2MetadataClientImplV1{client: metadata}
}

// InstanceIdentityDocument returns instance identity documents
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c *ec2MetadataClientImplV1) GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return c.client.GetInstanceIdentityDocument()
}

func (c *ec2MetadataClientImplV1) Region() (string, error) {
	return c.client.Region()
}

// HTTPClient is used to help with testing
type HTTPClient interface {
	GetInstanceIdentityDocument(ctx context.Context) (*v2ec2imds.GetInstanceIdentityDocumentOutput, error)
	GetRegion(ctx context.Context) (string, error)
}

// EC2MetadataClient is used to obtain a subset of information from EC2 IMDS
type EC2MetadataClient interface {
	GetInstanceIdentityDocument(ctx context.Context) (*v2ec2imds.GetInstanceIdentityDocumentOutput, error)
	GetRegion(ctx context.Context) (string, error)
}

type ec2MetadataClientImpl struct {
	client *v2ec2imds.Client
}

// New creates an ec2metadata client to retrieve metadata
func New(ctx context.Context) (EC2MetadataClient, error) {
	cfg, err := v2config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	imdsClient := v2ec2imds.NewFromConfig(cfg)
	return NewMetadataService(imdsClient), nil
}

// NewMetadataService creates an ec2metadata client to retrieve metadata
func NewMetadataService(client *v2ec2imds.Client) EC2MetadataClient {
	return &ec2MetadataClientImpl{client: client}
}

// GetInstanceIdentityDocument returns instance identity documents
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c *ec2MetadataClientImpl) GetInstanceIdentityDocument(ctx context.Context) (*v2ec2imds.GetInstanceIdentityDocumentOutput, error) {
	return c.client.GetInstanceIdentityDocument(ctx, &v2ec2imds.GetInstanceIdentityDocumentInput{})
}

func (c *ec2MetadataClientImpl) GetRegion(ctx context.Context) (string, error) {
	output, err := c.client.GetRegion(ctx, &v2ec2imds.GetRegionInput{})
	if err != nil {
		return "", err
	}
	return output.Region, nil
}
