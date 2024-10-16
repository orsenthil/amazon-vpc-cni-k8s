// Package ec2metadatawrapper is used to retrieve data from EC2 IMDS
package ec2metadatawrapper

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
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
