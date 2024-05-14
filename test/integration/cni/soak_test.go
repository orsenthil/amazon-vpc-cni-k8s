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

package cni

import (
	"github.com/aws/amazon-vpc-cni-k8s/test/framework/utils"
	"github.com/aws/amazon-vpc-cni-k8s/test/integration/common"
	"strconv"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/framework/resources/k8s/manifest"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
)

// Ensures Pods are launched on both Primary and Secondary Network Interfaces on two nodes.
// and the test verifies network connectivity across pods launched on these interfaces.

var _ = Describe("SOAK Test pod networking", func() {

	var (
		err error
		// The command to run on server pods, to allow incoming
		// connections for different traffic type
		serverListenCmd []string
		// Arguments to the server listen command
		serverListenCmdArgs []string
		// The function that generates command which will be sent from
		// tester pod to receiver pod
		testConnectionCommandFunc func(serverPod coreV1.Pod, port int) []string
		// The functions re-inforces that the positive test is working as
		// expected by creating a negative test command that should fail
		testFailedConnectionCommandFunc func(serverPod coreV1.Pod, port int) []string
		// Expected stdout from the exec command on testing connection
		// from tester to server
		testerExpectedStdOut string
		// Expected stderr from the exec command on testing connection
		// from tester to server
		testerExpectedStdErr string
		// The port on which server is listening for new connections
		serverPort int
		// Protocol for establishing connection to server
		protocol string

		// Primary node server deployment
		primaryNodeDeployment *v1.Deployment
		// Secondary node Server deployment
		secondaryNodeDeployment *v1.Deployment

		// Map of Pods placed on primary/secondary ENI IP on primary node
		interfaceToPodListOnPrimaryNode common.InterfaceTypeToPodList
		// Map of Pods placed on primary/secondary ENI IP on secondary node
		interfaceToPodListOnSecondaryNode common.InterfaceTypeToPodList

		// The total test will take 1 hour of constantly exercising pod launch on primary and secondary interfaces.
		// running connectivity tests, and deleting the pods, and repeating the process.
		// Any failure will fail the test
		timesToRunSoakTest  = 12
		waitDuringInMinutes = time.Duration(5) * time.Minute
	)

	JustBeforeEach(func() {
		By("authorizing security group ingress on instance security group")
		err = f.CloudServices.EC2().
			AuthorizeSecurityGroupIngress(instanceSecurityGroupID, protocol, serverPort, serverPort, "0.0.0.0/0", false)
		Expect(err).ToNot(HaveOccurred())

		By("authorizing security group egress on instance security group")
		err = f.CloudServices.EC2().
			AuthorizeSecurityGroupEgress(instanceSecurityGroupID, protocol, serverPort, serverPort, "0.0.0.0/0")
		Expect(err).ToNot(HaveOccurred())

		serverContainer := manifest.
			NewNetCatAlpineContainer(f.Options.TestImageRegistry).
			Command(serverListenCmd).
			Args(serverListenCmdArgs).
			Build()

		By("creating server deployment on the primary node")
		primaryNodeDeployment = manifest.
			NewDefaultDeploymentBuilder().
			Container(serverContainer).
			Replicas(maxIPPerInterface*2). // X2 so Pods are created on secondary ENI too
			NodeName(primaryNode.Name).
			PodLabel("node", "primary").
			Name("primary-node-server").
			Build()

		primaryNodeDeployment, err = f.K8sResourceManagers.
			DeploymentManager().
			CreateAndWaitTillDeploymentIsReady(primaryNodeDeployment, utils.DefaultDeploymentReadyTimeout)
		Expect(err).ToNot(HaveOccurred())

		interfaceToPodListOnPrimaryNode =
			common.GetPodsOnPrimaryAndSecondaryInterface(primaryNode, "node", "primary", f)

		// At least two Pods should be placed on the Primary and Secondary Interface
		// on the Primary and Secondary Node in order to test all possible scenarios
		Expect(len(interfaceToPodListOnPrimaryNode.PodsOnPrimaryENI)).
			Should(BeNumerically(">", 1))
		Expect(len(interfaceToPodListOnPrimaryNode.PodsOnSecondaryENI)).
			Should(BeNumerically(">", 1))

		By("creating server deployment on secondary node")
		secondaryNodeDeployment = manifest.
			NewDefaultDeploymentBuilder().
			Container(serverContainer).
			Replicas(maxIPPerInterface*2). // X2 so Pods are created on secondary ENI too
			NodeName(secondaryNode.Name).
			PodLabel("node", "secondary").
			Name("secondary-node-server").
			Build()

		secondaryNodeDeployment, err = f.K8sResourceManagers.
			DeploymentManager().
			CreateAndWaitTillDeploymentIsReady(secondaryNodeDeployment, utils.DefaultDeploymentReadyTimeout)
		Expect(err).ToNot(HaveOccurred())

		interfaceToPodListOnSecondaryNode =
			common.GetPodsOnPrimaryAndSecondaryInterface(secondaryNode, "node", "secondary", f)

		// Same reason as mentioned above
		Expect(len(interfaceToPodListOnSecondaryNode.PodsOnPrimaryENI)).
			Should(BeNumerically(">", 1))
		Expect(len(interfaceToPodListOnSecondaryNode.PodsOnSecondaryENI)).
			Should(BeNumerically(">", 1))
	})

	JustAfterEach(func() {
		By("revoking security group ingress on instance security group")
		err = f.CloudServices.EC2().
			RevokeSecurityGroupIngress(instanceSecurityGroupID, protocol, serverPort, serverPort, "0.0.0.0/0", false)
		Expect(err).ToNot(HaveOccurred())

		By("revoking security group egress on instance security group")
		err = f.CloudServices.EC2().
			RevokeSecurityGroupEgress(instanceSecurityGroupID, protocol, serverPort, serverPort, "0.0.0.0/0")
		Expect(err).ToNot(HaveOccurred())

		By("deleting the primary node server deployment")
		err = f.K8sResourceManagers.DeploymentManager().
			DeleteAndWaitTillDeploymentIsDeleted(primaryNodeDeployment)
		Expect(err).ToNot(HaveOccurred())

		By("deleting the secondary node server deployment")
		err = f.K8sResourceManagers.DeploymentManager().
			DeleteAndWaitTillDeploymentIsDeleted(secondaryNodeDeployment)
		Expect(err).ToNot(HaveOccurred())
	})

	for i := 0; i < timesToRunSoakTest; i++ {

		Context("[SOAK] when establishing TCP connection from tester to server", func() {

			BeforeEach(func() {
				serverPort = 2273
				protocol = ec2.ProtocolTcp
				// Test tcp connection using netcat
				serverListenCmd = []string{"nc"}
				// The nc flag "-l" for listen mode, "-k" to keep server up and not close
				// connection after each connection
				serverListenCmdArgs = []string{"-k", "-l", strconv.Itoa(serverPort)}

				// netcat verbose output is being redirected to stderr instead of stdout
				testerExpectedStdErr = "succeeded!"
				testerExpectedStdOut = ""

				// The nc flag "-v" for verbose output and "-wn" for timing out in n seconds
				testConnectionCommandFunc = func(receiverPod coreV1.Pod, port int) []string {
					return []string{"nc", "-v", "-w2", receiverPod.Status.PodIP, strconv.Itoa(port)}
				}

				// Create a negative test case with the wrong port number. This is to reinforce the
				// positive test case work by verifying negative cases do throw error
				testFailedConnectionCommandFunc = func(receiverPod coreV1.Pod, port int) []string {
					return []string{"nc", "-v", "-w2", receiverPod.Status.PodIP, strconv.Itoa(port + 1)}
				}
			})

			It("should allow connection across nodes and across interface types", func() {
				CheckConnectivityForMultiplePodPlacement(
					interfaceToPodListOnPrimaryNode, interfaceToPodListOnSecondaryNode,
					serverPort, testerExpectedStdOut, testerExpectedStdErr, testConnectionCommandFunc)

				By("verifying connection fails for unreachable port")
				VerifyConnectivityFailsForNegativeCase(interfaceToPodListOnPrimaryNode.PodsOnPrimaryENI[0],
					interfaceToPodListOnPrimaryNode.PodsOnPrimaryENI[1], serverPort,
					testFailedConnectionCommandFunc)
			})
		})

		time.Sleep(waitDuringInMinutes)

	}
})
