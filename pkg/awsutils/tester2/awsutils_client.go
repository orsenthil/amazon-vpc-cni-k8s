package main

import (
	"context"
	"fmt"
	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils" // Import for ENIMetadata type
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/sirupsen/logrus"
	"os"
)

type Client struct {
	imds awsutils.TypedIMDS
}

// NewClient creates a new IMDS client
func NewClient() *Client {
	return &Client{
		imds: awsutils.TypedIMDS{
			EC2MetadataIface: imds.New(imds.Options{}),
		},
	}
}

// GetAttachedENIs retrieves ENI information from instance metadata service
func (c *Client) GetAttachedENIs() ([]awsutils.ENIMetadata, error) {
	ctx := context.TODO()

	macs, err := c.imds.GetMACs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MACs: %v", err)
	}

	enis := make([]awsutils.ENIMetadata, len(macs))
	for i, mac := range macs {
		eni, err := c.getENIMetadata(mac)
		if err != nil {
			return nil, fmt.Errorf("failed to get ENI metadata for MAC %s: %v", mac, err)
		}
		enis[i] = eni
	}

	return enis, nil
}

func (cache *Client) getENIMetadata(eniMAC string) (awsutils.ENIMetadata, error) {
	ctx := context.TODO()

	log.Debugf("Found ENI MAC address: %s", eniMAC)
	var err error
	var deviceNum int

	eniID, err := cache.imds.GetInterfaceID(ctx, eniMAC)
	log.Debug("SENTHIL Getting ENI ID")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}

	log.Debugf("SENTHIL Found ENI ID address: %s", eniID)

	deviceNum, err = cache.imds.GetDeviceNumber(ctx, eniMAC)
	log.Debug("SENTHIL Getting Device Number")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}
	log.Debugf("SENTHIL Found Device Number: %d", deviceNum)

	primaryMAC, err := cache.imds.GetMAC(ctx)
	log.Debug("SENTHIL GETTING PRIMARY MAC")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}
	log.Debugf("SENTHIL Found primary MAC address: %s", primaryMAC)
	if eniMAC == primaryMAC && deviceNum != 0 {
		// Can this even happen? To be backwards compatible, we will always use 0 here and log an error.
		log.Errorf("Device number of primary ENI is %d! Forcing it to be 0 as expected", deviceNum)
		deviceNum = 0
	}

	log.Debugf("Found ENI: %s, MAC %s, device %d", eniID, eniMAC, deviceNum)

	// Get IMDS fields for the interface
	macImdsFields, err := cache.imds.GetMACImdsFields(ctx, eniMAC)
	log.Debug("Getting MAC IMDS Fields")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}
	log.Debugf("Found IMDS fields for interface: %v", macImdsFields)
	ipInfoAvailable := false
	// Efa-only interfaces do not have any ipv4s or ipv6s associated with it. If we don't find any local-ipv4 or ipv6 info in imds we assume it to be efa-only interface and validate this later via ec2 call
	for _, field := range macImdsFields {
		if field == "local-ipv4s" {
			imdsIPv4s, err := cache.imds.GetLocalIPv4s(ctx, eniMAC)
			if err != nil {
				return awsutils.ENIMetadata{}, err
			}
			log.Debugf("Found local imds IPv4 addresses: %v", imdsIPv4s)
			if len(imdsIPv4s) > 0 {
				ipInfoAvailable = true
				log.Debugf("Found IPv4 addresses associated with interface. This is not efa-only interface")
				break
			}
		}
		if field == "ipv6s" {
			imdsIPv6s, err := cache.imds.GetIPv6s(ctx, eniMAC)
			if err != nil {
				// awsAPIErrInc("GetIPv6s", err)
			} else if len(imdsIPv6s) > 0 {
				ipInfoAvailable = true
				log.Debugf("Found IPv6 addresses associated with interface. This is not efa-only interface")
				break
			}
		}
	}

	if !ipInfoAvailable {
		return awsutils.ENIMetadata{
			ENIID:          eniID,
			MAC:            eniMAC,
			DeviceNumber:   deviceNum,
			SubnetIPv4CIDR: "",
			IPv4Addresses:  make([]ec2types.NetworkInterfacePrivateIpAddress, 0),
			IPv4Prefixes:   make([]ec2types.Ipv4PrefixSpecification, 0),
			SubnetIPv6CIDR: "",
			IPv6Addresses:  make([]ec2types.NetworkInterfaceIpv6Address, 0),
			IPv6Prefixes:   make([]ec2types.Ipv6PrefixSpecification, 0),
		}, nil
	}

	// Get IPv4 and IPv6 addresses assigned to interface
	cidr, err := cache.imds.GetSubnetIPv4CIDRBlock(ctx, eniMAC)
	log.Debug("FINDING SUBNET CIDR")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}
	log.Debugf("FOUND SUBNET CIDR: %s", cidr)

	imdsIPv4s, err := cache.imds.GetLocalIPv4s(ctx, eniMAC)
	log.Debug("Finding Local IPv4s")
	if err != nil {
		return awsutils.ENIMetadata{}, err
	}
	log.Debugf("Found local imds IPv4 addresses: %v", imdsIPv4s)

	ec2ip4s := make([]ec2types.NetworkInterfacePrivateIpAddress, len(imdsIPv4s))
	log.Debugf("Found ec2 IPv4 addresses: %v", ec2ip4s)
	log.Debug("Finding ec2 IPv4s")
	log.Debug("SENTHIL - Finding a Problem after this statement.")
	for i, ip4 := range imdsIPv4s {
		ec2ip4s[i] = ec2types.NetworkInterfacePrivateIpAddress{
			Primary:          aws.Bool(i == 0),
			PrivateIpAddress: aws.String(ip4.String()),
		}
	}
	log.Debugf("Found ec2 IPv4 addresses: %v", ec2ip4s)

	for _, ec2ipv4 := range ec2ip4s {
		log.Debugf("private ip %v", *ec2ipv4.PrivateIpAddress)
	}

	var ec2ip6s []ec2types.NetworkInterfaceIpv6Address
	var subnetV6Cidr string

	var ec2ipv4Prefixes []ec2types.Ipv4PrefixSpecification
	var ec2ipv6Prefixes []ec2types.Ipv6PrefixSpecification

	log.Debugf("Getting Prefixes")
	// If IPv6 is enabled, get attached v6 prefixes.
	if eniMAC != primaryMAC {
		// Get prefix on primary ENI when custom networking is enabled is not needed.
		// If primary ENI has prefixes attached and then we move to custom networking, we don't need to fetch
		// the prefix since recommendation is to terminate the nodes and that would have deleted the prefix on the
		// primary ENI.
		imdsIPv4Prefixes, err := cache.imds.GetIPv4Prefixes(ctx, eniMAC)
		log.Debugf("Getting IPV4 Prefixes")
		if err != nil {
			return awsutils.ENIMetadata{}, err
		}
		log.Debugf("Completed getting IPV4 Prefixes.")
		for _, ipv4prefix := range imdsIPv4Prefixes {
			ec2ipv4Prefixes = append(ec2ipv4Prefixes, ec2types.Ipv4PrefixSpecification{
				Ipv4Prefix: aws.String(ipv4prefix.String()),
			})
		}
	}
	log.Debugf("Found ec2 IPv4 prefixes: %v", ec2ipv4Prefixes)

	return awsutils.ENIMetadata{
		ENIID:          eniID,
		MAC:            eniMAC,
		DeviceNumber:   deviceNum,
		SubnetIPv4CIDR: cidr.String(),
		IPv4Addresses:  ec2ip4s,
		IPv4Prefixes:   ec2ipv4Prefixes,
		SubnetIPv6CIDR: subnetV6Cidr,
		IPv6Addresses:  ec2ip6s,
		IPv6Prefixes:   ec2ipv6Prefixes,
	}, nil
}

func main() {
	client := NewClient()
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	enis, err := client.GetAttachedENIs()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d ENIs:\n", len(enis))
	for _, eni := range enis {
		fmt.Printf("\nENI ID: %s\n", eni.ENIID)
		fmt.Printf("MAC: %s\n", eni.MAC)
		fmt.Printf("Device Number: %d\n", eni.DeviceNumber)
		fmt.Printf("Subnet IPv4 CIDR: %s\n", eni.SubnetIPv4CIDR)

		fmt.Println("IPv4 Addresses:")
		for _, addr := range eni.IPv4Addresses {
			fmt.Printf("  %s (Primary: %v)\n", *addr.PrivateIpAddress, *addr.Primary)
		}

		if len(eni.IPv6Addresses) > 0 {
			fmt.Printf("Subnet IPv6 CIDR: %s\n", eni.SubnetIPv6CIDR)
			fmt.Println("IPv6 Addresses:")
			for _, addr := range eni.IPv6Addresses {
				fmt.Printf("  %s\n", *addr.Ipv6Address)
			}
		}
	}
}
