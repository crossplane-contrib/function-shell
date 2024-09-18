# Example - AWS

This example demonstrates how to reuse an existing `ProviderConfig` to authenticate with AWS using IAM Role for
ServiceAccount (IRSA) and execute a command on a different account using AssumeRole. This architecture is known as Hub
and Spoke.

## Prerequisites

### Authenticate using IAM Roles for Service Accounts

The Amazon Elastic Kubernetes Service (EKS) running the function will need a ServiceAccount authorized to authenticate
with AWS. The steps are details in the configuration of 
[provider-upjet-aws](https://github.com/crossplane-contrib/provider-upjet-aws/blob/main/docs/family/Configuration.md#authenticate-using-iam-roles-for-service-accounts).

### Authorize Hub account on the Spoke Account

The previous step has created the AWS IAM Role`arn:aws:iam::000000000000:role/eks-test-role`. It now needs to be allowed
to make requests on the target account.

#### Create an IAM policy

Define the actions allowed on the target account. For this example, it will be restricted to listing IAM Roles. Adapt
the policy based on your needs.

Apply the policy using the AWS command-line command:
```bash
aws iam create-policy \
    --policy-name IAMRoleLister \
    --policy-document \
'{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "iam:ListRoles"
            ],
            "Resource": "*",
            "Effect": "Allow"
        }
    ]
}'
```

Example output:
```json
{
    "Policy": {
        "PolicyName": "IAMRoleLister",
        "PolicyId": "ANPAS43KCNAZNTZIWNOCU",
        "Arn": "arn:aws:iam::000000000001:policy/IAMRoleLister",
        "Path": "/",
        "DefaultVersionId": "v1",
        "AttachmentCount": 0,
        "PermissionsBoundaryUsageCount": 0,
        "IsAttachable": true,
        "CreateDate": "2024-09-17T08:35:55+00:00",
        "UpdateDate": "2024-09-17T08:35:55+00:00"
    }
}
```

#### Create an IAM role

Define the trust policy to allow the Hub account (000000000000) to AssumeRole on the target account (000000000001).

```bash
aws iam create-role \
    --role-name eks-test-assume-role \
    --assume-role-policy-document \
'{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::000000000000:role/eks-test-role"
            },
            "Action": "sts:AssumeRole",
            "Condition": {}
        }
    ]
}'
```

Example output:
```json
{
    "Role": {
        "Path": "/",
        "RoleName": "eks-test-assume-role",
        "RoleId": "AROAS43KCNAZGWLU4RRHM",
        "Arn": "arn:aws:iam::000000000001:role/eks-test-assume-role",
        "CreateDate": "2024-09-17T08:48:30+00:00",
        "AssumeRolePolicyDocument": {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Principal": {
                        "AWS": "arn:aws:iam::000000000000:role/eks-test-role"
                    },
                    "Action": "sts:AssumeRole",
                    "Condition": {}
                }
            ]
        }
    }
}
```

Attach the IAMRoleLister policy on the role

```bash
aws iam attach-role-policy \
    --policy-arn arn:aws:iam::000000000001:policy/IAMRoleLister \
    --role-name eks-test-assume-role
 ```

## Usage

### Loading ProviderConfig with function-extra-resources

The composition pipeline use `function-extra-resources` to load ProviderConfig defined in the EKS cluster. For this example
the following has been applied:

```yaml
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  labels:
    account: demo
  name: demo
spec:
  assumeRoleChain:
    - roleARN: arn:aws:iam::000000000001:role/eks-test-assume-role
  credentials:
    source: IRSA
```

The IAM role of the target account is placed at `spec.assumeRoleChain[0].roleARN`

The `function-extra-resources` use a selector to discover `ProviderConfig`, `maxMatch: 1` and `minMatch: 1`have been set
so that exactly one `ProviderConfig` is return. In case none is found the composition will error out.

### Configuring function-shell

AWS CLI is configured with the following environment variable
- **AWS_ROLE_ARN** : is the role created on the hub account
- **AWS_ASSUME_ROLE_ARN**: is the role on the targeted account 

The **AWS_ASSUME_ROLE_ARN** is configured dynamically with `valueRef` with the following expression:
`context[apiextensions.crossplane.io/extra-resources].ProviderConfig[0].spec.assumeRoleChain[0].roleARN`

The reference is retrieving from the context at the `apiextensions.crossplane.io/extra-resources`key. In the previous step
the `ProviderConfig` are saved into the `ProviderConfig` key (set with `function-extra-resources` input `spec.extraResources[0].into`)

For the function to be authenticated it first request a temporary token using:
```bash
ASSUME_ROLE_OUTPUT=$(aws sts assume-role --role-arn $AWS_ASSUME_ROLE_ARN --role-session-name "function-shell")
```

The session name can be set to an arbitrary value; it enables keeping track of the service making the call.

The environment variable `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_SESSION_TOKEN` are set from the output with:
```bash
export AWS_ACCESS_KEY_ID=$(echo $ASSUME_ROLE_OUTPUT | grep -o '"AccessKeyId": "[^"]*"' | cut -d'"' -f4)
export AWS_SECRET_ACCESS_KEY=$(echo $ASSUME_ROLE_OUTPUT | grep -o '"SecretAccessKey": "[^"]*"' | cut -d'"' -f4)
export AWS_SESSION_TOKEN=$(echo $ASSUME_ROLE_OUTPUT | grep -o '"SessionToken": "[^"]*"' | cut -d'"' -f4)
```

Finally, the AWS command is executed and filtered with `jq` to retrieve the ARN of all the roles in the target account with:
```bash
aws iam list-roles | jq -r '.Roles[] .Arn'
```
