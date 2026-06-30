#this terraform code makes a trust policy for an ec2 service to assume the role of ssm_role_ec2 this role has the policy attachment AmazonSSMManagedInstanceCore which lets it connect to SSM, we then make the instance profile which we will later attach to the VM


data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "ssm_role" {
  name               = "ssm_role_ec2"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}


resource "aws_iam_role_policy_attachment" "attach_ssm_policy_to_role" {
  role       = aws_iam_role.role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "ec2_ssm_profile" {
  name = "ssm_profile"
  role = aws_iam_role.role.name
}


