#!/bin/sh

echo "The arguments passed to this script are: $@"

SECRET1_USERNAME_PATH=$(cat "$SECRET1_USERNAME_PATH")
SECRET1_PASSWORD_PATH=$(cat "$SECRET1_PASSWORD_PATH")
SECRET2_USERNAME_PATH=$(cat "$SECRET2_USERNAME_PATH")
SECRET2_PASSWORD_PATH=$(cat "$SECRET2_PASSWORD_PATH")

while true; do
  echo "======================================"
  echo "secret1_username_mountfile: $SECRET1_USERNAME_PATH"
  echo "secret1_password_mountfile: $SECRET1_PASSWORD_PATH"
  echo "secret2_username_mountfile: $SECRET2_USERNAME_PATH"
  echo "secret2_password_mountfile: $SECRET2_PASSWORD_PATH"
  echo "======================================"
  echo "secret1_username_env: $USERNAME1"
  echo "secret1_password_env: $PASSWORD1"
  echo "secret2_username_env: $USERNAME2"
  echo "secret2_password_env: $PASSWORD2"
  echo "======================================"
  sleep 10
done
