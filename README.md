### To Deploy
From broker/ -> run:
    ./scripts/deploy.sh

### Run To Generate The Certs
sudo certbot certonly --standalone -d fund78.com -d www.fund78.com --dry-run -vvv
sudo certbot certonly --standalone -d fund78.com -d www.fund78.com

### Where The Certs Are Found On The Server
/etc/letsencrypt/live/fund78.com

### Run Website Locally
cd /frontend
npm run build