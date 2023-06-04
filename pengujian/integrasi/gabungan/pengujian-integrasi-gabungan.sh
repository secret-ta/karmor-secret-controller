#!/bin/sh

echo "The arguments passed to this script are: $@"

SECRET_V1_USERNAME_PATH=$(cat "$SECRET_V1_USERNAME_PATH")
SECRET_V1_PASSWORD_PATH=$(cat "$SECRET_V1_PASSWORD_PATH")
SECRET_V2_USERNAME_PATH=$(cat "$SECRET_V2_USERNAME_PATH")
SECRET_V2_PASSWORD_PATH=$(cat "$SECRET_V2_PASSWORD_PATH")

while true; do
  echo "======================================"
  echo "secret_v1_username_mountfile: $SECRET_V1_USERNAME_PATH"
  echo "secret_v1_password_mountfile: $SECRET_V1_PASSWORD_PATH"
  echo "secret_v2_username_mountfile: $SECRET_V2_USERNAME_PATH"
  echo "secret_v2_password_mountfile: $SECRET_V2_PASSWORD_PATH"
  echo "======================================"
  echo "secret_v1_username_env: $USERNAME_V1"
  echo "secret_v1_password_env: $PASSWORD_V1"
  echo "secret_v2_username_env: $USERNAME_V2"
  echo "secret_v2_password_env: $PASSWORD_V2"
  echo "======================================"
  sleep 10
done
