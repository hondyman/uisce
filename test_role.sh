#!/bin/bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFkbWluQHVpc2NlLmNvbSIsImV4cCI6MTc2NzI0Njk0MCwiaHR0cHM6Ly9oYXN1cmEuaW8vand0L2NsYWltcyI6eyJ4LWhhc3VyYS1hbGxvd2VkLXJvbGVzIjpbInVzZXIiLCJnbG9iYWxfYWRtaW4iLCJhZG1pbiJdLCJ4LWhhc3VyYS1kZWZhdWx0LXJvbGUiOiJnbG9iYWxfYWRtaW4iLCJ4LWhhc3VyYS11c2VyLWlkIjoiOTkwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIn0sImlhdCI6MTc2NzI0MzM0MCwiaXNfY29yZV9hZG1pbiI6dHJ1ZSwibmFtZSI6Ikdsb2JhbCBBZG1pbiIsIm9yZ2FuaXphdGlvbiI6InVpc2NlIiwicGVybWlzc2lvbnMiOltdLCJyb2xlIjoiYWRtaW4iLCJ0ZW5hbnRfaWQiOiIiLCJ1c2VyX2lkIjoiOTkwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIn0.c8Ta0X1cmtpdihiJl6y3edlXPzWVa71DYZTdLliCrRo"

echo "Testing Role Creation..."
curl -v -X POST http://127.0.0.1:8080/api/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "test_analyst_04",
    "description": "Fourth test role",
    "is_global_admin": false
  }' > curl_output.txt 2>&1

echo "Curl Exit Code: $?"
cat curl_output.txt
