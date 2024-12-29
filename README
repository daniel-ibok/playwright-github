# GOLANG PLAYWRIGHT GITHUB LOGIN API AUTOMATOR
provide github credentials to api endpoint to automate github login

## Installatiaon
Clone the repository:
```bash
 git clone https://github.com/daniel-ibok/playwright-github.git
```
## Usage
To run the project, use the following command:
```bash
  go run main.go
```
Make request to login api endpoint with your github credentials
```bash
  curl -X POST http://locahost:9001/login -d email="yourusername" password="password"
```
After successfully response from the above request, provide totp code for authentication to complete login
```bash
  curl -X POST http://locahost:9001/twoauth -d code="yourtotpcode"
```

The response from the above request returns yout github cookie and username
