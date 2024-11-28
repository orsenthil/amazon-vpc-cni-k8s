package main

import (
	"context"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"net"
)

type Client struct {
	imds awsutils.TypedIMDS
}

func (c *Client) GetAZ(ctx context.Context) (string, error) {
	return c.imds.GetAZ(ctx)
}

func (c *Client) GetInstanceType(ctx context.Context) (string, error) {
	return c.imds.GetInstanceType(ctx)
}

func (c *Client) GetLocalIPv4(ctx context.Context) (net.IP, error) {
	return c.imds.GetLocalIPv4(ctx)
}

func (c *Client) GetInstanceID(ctx context.Context) (string, error) {
	return c.imds.GetInstanceID(ctx)
}

func (c *Client) GetMAC(ctx context.Context) (string, error) {
	return c.imds.GetMAC(ctx)
}

func (c *Client) GetMACs(ctx context.Context) ([]string, error) {
	return c.imds.GetMACs(ctx)
}

func (c *Client) GetMACImdsFields(ctx context.Context, mac string) ([]string, error) {
	return c.imds.GetMACImdsFields(ctx, mac)
}

func (c *Client) GetInterfaceID(ctx context.Context, mac string) (string, error) {
	return c.imds.GetInterfaceID(ctx, mac)
}

func (c *Client) GetDeviceNumber(ctx context.Context, mac string) (int, error) {
	return c.imds.GetDeviceNumber(ctx, mac)
}

func (c *Client) GetSubnetID(ctx context.Context, mac string) (string, error) {
	return c.imds.GetSubnetID(ctx, mac)
}

func (c *Client) GetVpcID(ctx context.Context, mac string) (string, error) {
	return c.imds.GetVpcID(ctx, mac)
}

func (c *Client) GetSecurityGroupIDs(ctx context.Context, mac string) ([]string, error) {
	return c.imds.GetSecurityGroupIDs(ctx, mac)
}

func (c *Client) GetLocalIPv4s(ctx context.Context, mac string) ([]net.IP, error) {
	return c.imds.GetLocalIPv4s(ctx, mac)
}

func (c *Client) GetIPv4Prefixes(ctx context.Context, mac string) ([]net.IPNet, error) {
	return c.imds.GetIPv4Prefixes(ctx, mac)
}

func (c *Client) GetIPv6Prefixes(ctx context.Context, mac string) ([]net.IPNet, error) {
	return c.imds.GetIPv6Prefixes(ctx, mac)
}

func (c *Client) GetIPv6s(ctx context.Context, mac string) ([]net.IP, error) {
	return c.imds.GetIPv6s(ctx, mac)
}

func (c *Client) GetSubnetIPv4CIDRBlock(ctx context.Context, mac string) (net.IPNet, error) {
	return c.imds.GetSubnetIPv4CIDRBlock(ctx, mac)
}

func (c *Client) GetVPCIPv4CIDRBlocks(ctx context.Context, mac string) ([]net.IPNet, error) {
	return c.imds.GetVPCIPv4CIDRBlocks(ctx, mac)
}

func (c *Client) GetVPCIPv6CIDRBlocks(ctx context.Context, mac string) ([]net.IPNet, error) {
	return c.imds.GetVPCIPv6CIDRBlocks(ctx, mac)
}

func (c *Client) GetSubnetIPv6CIDRBlocks(ctx context.Context, mac string) (net.IPNet, error) {
	return c.imds.GetSubnetIPv6CIDRBlocks(ctx, mac)
}

// NewClient creates a new IMDS client
func NewClient() *Client {
	return &Client{
		imds: awsutils.TypedIMDS{
			EC2MetadataIface: imds.New(imds.Options{}),
		},
	}
}

func main() {
	// Example usage
	client := NewClient()
	ctx := context.Background()

	// Example usage of all methods
	if az, err := client.GetAZ(ctx); err == nil {
		println("AZ:", az)
	}

	if instanceType, err := client.GetInstanceType(ctx); err == nil {
		println("Instance Type:", instanceType)
	}

	if ip, err := client.GetLocalIPv4(ctx); err == nil {
		println("Local IPv4:", ip.String())
	}

	if instanceID, err := client.GetInstanceID(ctx); err == nil {
		println("Instance ID:", instanceID)
	}

	// Get all MACs and their associated info
	macs, err := client.GetMACs(ctx)
	if err != nil {
		panic(err)
	}

	for _, mac := range macs {
		println("\nInterface MAC:", mac)

		if id, err := client.GetInterfaceID(ctx, mac); err == nil {
			println("Interface ID:", id)
		}

		if num, err := client.GetDeviceNumber(ctx, mac); err == nil {
			println("Device Number:", num)
		}

		if subnet, err := client.GetSubnetID(ctx, mac); err == nil {
			println("Subnet ID:", subnet)
		}

		if vpc, err := client.GetVpcID(ctx, mac); err == nil {
			println("VPC ID:", vpc)
		}

		if sgs, err := client.GetSecurityGroupIDs(ctx, mac); err == nil {
			println("Security Groups:", sgs)
		}

		if ips, err := client.GetLocalIPv4s(ctx, mac); err == nil {
			for _, ip := range ips {
				println("Local IPv4:", ip.String())
			}
		}

		if prefixes, err := client.GetIPv4Prefixes(ctx, mac); err == nil {
			for _, prefix := range prefixes {
				println("IPv4 Prefix:", prefix.String())
			}
		}

		if ips, err := client.GetIPv6s(ctx, mac); err == nil {
			for _, ip := range ips {
				println("IPv6:", ip.String())
			}
		}

		if prefixes, err := client.GetIPv6Prefixes(ctx, mac); err == nil {
			for _, prefix := range prefixes {
				println("IPv6 Prefix:", prefix.String())
			}
		}
	}
}
