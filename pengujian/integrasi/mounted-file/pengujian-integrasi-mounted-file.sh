#!/bin/sh

echo "The arguments passed to this script are: $@"

SECRET_V1_USERNAME_PATH=$(cat "$SECRET_V1_USERNAME_PATH")
SECRET_V1_PASSWORD_PATH=$(cat "$SECRET_V1_PASSWORD_PATH")
SECRET_V2_USERNAME_PATH=$(cat "$SECRET_V2_USERNAME_PATH")
SECRET_V2_PASSWORD_PATH=$(cat "$SECRET_V2_PASSWORD_PATH")

while true; do
  echo "secret_v1_username: $SECRET_V1_USERNAME_PATH"
  echo "secret_v1_password: $SECRET_V1_PASSWORD_PATH"
  echo "secret_v2_username: $SECRET_V2_USERNAME_PATH"
  echo "secret_v2_password: $SECRET_V2_PASSWORD_PATH"
  sleep 10
done
