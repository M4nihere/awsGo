package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

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
	fmt.Println("AWS_REGION:", AWS_REGION)
	fmt.Println("AWS_ACCESS_KEY_ID:", AWS_ACCESS_KEY_ID)
	fmt.Println("AWS_SECRET_ACCESS_KEY:", AWS_SECRET_ACCESS_KEY)

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
