"""Resolve the image once during preparation, never during a launch retry."""
import re
import subprocess

PARAMETER = '/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64'


def resolve_image(image_id, region, runner=subprocess.run):
    if image_id is None:
        try:
            result = runner([
                'aws', 'ssm', 'get-parameter', '--region', region,
                '--name', PARAMETER, '--query', 'Parameter.Value', '--output', 'text',
            ], check=True, capture_output=True, text=True, timeout=30)
        except subprocess.CalledProcessError as error:
            raise ValueError('AMI lookup failed: ' + (error.stderr or str(error)).strip()) from error
        image_id = result.stdout.strip()
    if not re.fullmatch(r'ami-(?:[0-9a-f]{8}|[0-9a-f]{17})', image_id):
        raise ValueError('image must be a concrete AMI ID; moving resolve:ssm references are not accepted')
    return image_id
