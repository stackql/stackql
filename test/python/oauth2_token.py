import base64, json

import argparse


def create_token(header: dict, claims: dict) -> str:
    """
    Create a token from the claims.

    TODO: Implement the signature.
    """
    header_b64 = base64.urlsafe_b64encode(json.dumps(header, sort_keys=True).encode('utf-8')).decode('utf-8')
    claims_b64 = base64.urlsafe_b64encode(json.dumps(claims, sort_keys=True).encode('utf-8')).decode('utf-8')
    signature = f'{header_b64}.{claims_b64}' # placeholder rubbish
    return f'{header_b64}.{claims_b64}.{base64.urlsafe_b64encode(signature.encode("utf-8")).decode("utf-8")}'


def parse_args() -> argparse.Namespace:
    """
    Parse the arguments.
    """
    parser = argparse.ArgumentParser(description='Create a token.')
    parser.add_argument('--create-token', help='Opt-in create token', action=argparse.BooleanOptionalAction)
    parser.add_argument('--header', type=str, help='The header.')
    parser.add_argument('--claims', type=str, help='The claims.')
    return parser.parse_args()


def generate_token(ns: argparse.Namespace) -> str:
    """
    Create a token.
    """
    header = json.loads(ns.header)
    claims = json.loads(ns.claims)
    return create_token(header, claims)


def main() -> None:
    """
    Main entry point.
    """
    args = parse_args()
    if args.create_token:
        print(generate_token(args))
        return
    exit(1)


if __name__ == '__main__':
    main()

