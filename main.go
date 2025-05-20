package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/joho/godotenv"
)

func main() {
	// Define flag for .env file path
	envFile := flag.String("env", ".env", "Path to .env file")
	flag.Parse()

	// Check if the .env file exists
	if _, err := os.Stat(*envFile); os.IsNotExist(err) {
		fmt.Printf("Info: .env file does not exist at: %s. Falling back to system environment variables.\n", *envFile)
	} else {
		// Load environment variables from .env file
		if err := godotenv.Load(*envFile); err != nil {
			log.Printf("Warning: Could not load .env file: %v. Falling back to system environment variables.", err)
		} else {
			fmt.Printf("Using .env file at: %s\n", *envFile)
		}
	}
	AWS_REGION := os.Getenv("AWS_REGION")
	if AWS_REGION == "" {
		log.Fatal("Error: AWS_REGION environment variable is not set.")

	}
	AWS_ACCESS_KEY_ID := os.Getenv("AWS_ACCESS_KEY_ID")
	if AWS_ACCESS_KEY_ID == "" {
		log.Fatal("Error: AWS_ACCESS_KEY_ID environment variable is not set.")
	}
	AWS_SECRET_ACCESS_KEY := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if AWS_SECRET_ACCESS_KEY == "" {
		log.Fatal("Error: AWS_SECRET_ACCESS_KEY environment variable is not set.")
	}
	// fmt.Println("AWS_REGION:", AWS_REGION)
	// fmt.Println("AWS_ACCESS_KEY_ID:", AWS_ACCESS_KEY_ID)
	// fmt.Println("AWS_SECRET_ACCESS_KEY:", AWS_SECRET_ACCESS_KEY)

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWS_REGION))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	ec2Client := ec2.NewFromConfig(cfg)

	// Get default VPC
	vpcID, err := getDefaultVPC(ec2Client)
	if err != nil {
		log.Fatalf("Failed to get default VPC: %v", err)
	}
	fmt.Printf("Using Default VPC ID: %s\n", *vpcID)

	ElasticIP, err := allocateElasticIP(ec2Client, *vpcID)
	if err != nil {
		log.Fatalf("Failed to allocate Elastic IP: %v", err)
	} else {
		fmt.Printf("Elastic IP: %s\n", *ElasticIP)
	}

	// Create a Key Pair from the public key
	err = createKeyPairFromPublicKey(ec2Client)
	if err != nil {
		log.Fatalf("Failed to create key pair from public key: %v", err)
	} else {
		fmt.Println("Key pair created successfully.")
	}
	// Create a new EC2 instance
	ec2instance, err := createEC2Instance(ec2Client)
	if err != nil {
		log.Fatalf("Failed to create EC2 instance: %v", err)
	} else {
		fmt.Printf("EC2 Instance ID: %s\n", *ec2instance)
	}

}

func getDefaultVPC(client *ec2.Client) (*string, error) {
	vpcs, err := client.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   awsString("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(vpcs.Vpcs) == 0 {
		return nil, fmt.Errorf("no default VPC found")
	}
	return vpcs.Vpcs[0].VpcId, nil
}

func allocateElasticIP(client *ec2.Client, vpcID string) (*string, error) {
	// Allocate an Elastic IP address
	eip, err := client.AllocateAddress(context.TODO(), &ec2.AllocateAddressInput{
		Domain: types.DomainTypeVpc,
	})
	if err != nil {
		return nil, err
	}
	return eip.PublicIp, nil

}

// func createEC2Instance(client *ec2.Client, inputSubnetID string) (*string, error) {
// 	// Retrieve environment variables
// 	amiID := os.Getenv("AMI_ID")
// 	if amiID == "" {
// 		return nil, fmt.Errorf("AMI_ID environment variable is not set")
// 	}

// 	instanceType := os.Getenv("INSTANCE_TYPE")
// 	if instanceType == "" {
// 		return nil, fmt.Errorf("INSTANCE_TYPE environment variable is not set")
// 	}

// 	keyName := os.Getenv("KEY_NAME")
// 	if keyName == "" {
// 		return nil, fmt.Errorf("KEY_NAME environment variable is not set")
// 	}

// 	// Describe subnets to validate or find a default subnet
// 	subnets, err := client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to describe subnets: %w", err)
// 	}

// 	if len(subnets.Subnets) == 0 {
// 		return nil, fmt.Errorf("no subnets found in the current VPC")
// 	}

// 	subnetID := inputSubnetID
// 	validSubnetID := false

// 	// Validate the provided subnet ID
// 	for _, subnet := range subnets.Subnets {
// 		if inputSubnetID != "" && *subnet.SubnetId == inputSubnetID {
// 			validSubnetID = true
// 			break
// 		}
// 	}

// 	// Default to the first available subnet if the provided one is invalid or empty
// 	if !validSubnetID {
// 		fmt.Printf("Provided subnet ID %s is invalid. Using the first available subnet: %s\n", inputSubnetID, *subnets.Subnets[0].SubnetId)
// 		subnetID = *subnets.Subnets[0].SubnetId
// 	}

// 	// Create an EC2 instance
// 	output, err := client.RunInstances(context.TODO(), &ec2.RunInstancesInput{
// 		ImageId:      awsString(amiID),
// 		InstanceType: types.InstanceType(instanceType),
// 		MinCount:     awsInt32(1),
// 		MaxCount:     awsInt32(1),
// 		SubnetId:     awsString(subnetID),
// 		KeyName:      awsString(keyName),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(output.Instances) == 0 {
// 		return nil, fmt.Errorf("no instances were created")
// 	}

//		return output.Instances[0].InstanceId, nil
//	}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}
func createKeyPairFromPublicKey(client *ec2.Client) error {

	keyName := os.Getenv("KEY_NAME")
	if keyName == "" {
		return fmt.Errorf("KEY_NAME environment variable is not set")
	}
	publicKeyPath := os.Getenv("PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		return fmt.Errorf("PUBLIC_KEY_PATH environment variable is not set")
	}
	// Expand ~ to home directory
	publicKeyPath, err := expandPath(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to expand public key path: %w", err)
	}

	// Read the public key file content
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}
	// Import the key pair to AWS using the public key material
	_, err = client.ImportKeyPair(context.TODO(), &ec2.ImportKeyPairInput{
		KeyName:           awsString(keyName),
		PublicKeyMaterial: publicKeyData,
	})
	if err != nil {
		return fmt.Errorf("failed to import key pair: %w", err)
	}

	fmt.Printf("Key pair '%s' created successfully from public key file.\n", keyName)
	return nil
}

func createEC2Instance(client *ec2.Client) (*string, error) {
	// Retrieve environment variables
	amiID := os.Getenv("AMI_ID")
	if amiID == "" {
		return nil, fmt.Errorf("AMI_ID environment variable is not set")
	}

	instanceType := os.Getenv("INSTANCE_TYPE")
	if instanceType == "" {
		return nil, fmt.Errorf("INSTANCE_TYPE environment variable is not set")
	}

	keyName := os.Getenv("KEY_NAME")
	if keyName == "" {
		return nil, fmt.Errorf("KEY_NAME environment variable is not set")
	}

	// Describe subnets to find a default subnet
	subnets, err := client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(subnets.Subnets) == 0 {
		return nil, fmt.Errorf("no subnets found in the current VPC")
	}

	// Use the first available subnet
	subnetID := *subnets.Subnets[0].SubnetId
	fmt.Printf("Using subnet ID: %s\n", subnetID)

	// Create an EC2 instance
	output, err := client.RunInstances(context.TODO(), &ec2.RunInstancesInput{
		ImageId:      awsString(amiID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     awsInt32(1),
		MaxCount:     awsInt32(1),
		SubnetId:     awsString(subnetID),
		KeyName:      awsString(keyName),
	})
	if err != nil {
		return nil, err
	}

	if len(output.Instances) == 0 {
		return nil, fmt.Errorf("no instances were created")
	}

	return output.Instances[0].InstanceId, nil
}

func awsString(value string) *string {
	return &value
}

func awsInt64(value int64) *int64 {
	return &value
}

func awsInt32(value int32) *int32 {
	return &value
}
