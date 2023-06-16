# basic 
# docker run -d -p 5000:5000 --restart=always --name registry registry:2
#https://docs.docker.com/registry/deploying/#run-the-registry-as-a-service


# windows create auth/htpasswd
docker run --rm --entrypoint htpasswd httpd:2 -Bbn testuser testpassword | Set-Content -Encoding ASCII auth/htpasswd
# linux create auth/htpasswd
mkdir auth
docker run \
  --entrypoint htpasswd \
  httpd:2 -Bbn luo ruofeng > auth/htpasswd

# certs
# sudo mkdir /certs


# use-self-signed-certificates：https://docs.docker.com/registry/insecure/#use-self-signed-certificates
# sudo openssl req \
#   -newkey rsa:4096 -nodes -sha256 -keyout ./certs/registry.luoruofeng.local.key \
#   -addext "subjectAltName = DNS:registry.luoruofeng.local" \
#   -x509 -days 365 -out ./certs/registry.luoruofeng.local.crt

# TLS and Auth
# docker run -d   -p 5000:5000 -p 443:443 --restart=always   --name registry   -v "$(pwd)"/auth:/auth   -e "REGISTRY_AUTH=htpasswd"   -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm"   -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd   -v "$(pwd)"/certs:/certs   -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/cert.pem   -e REGISTRY_HTTP_TLS_KEY=/certs/key.pem -v "${PWD}"/registry_data:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:443  registry:2

# TLS without Auth
docker run -d -p 443:443 -p 5000:5000 --restart=always   --name registry   -v "$(pwd)"/certs:/certs   -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt   -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key -v "${PWD}"/registry_data:/var/lib/registry --restart=always -e REGISTRY_HTTP_ADDR=0.0.0.0:443  registry:2

#API query catalog
curl --cert certs/registry.luoruofeng.local.crt --key certs/registry.luoruofeng.local.key --insecure https://registry.luoruofeng.local:443/v2/_catalog
# or
 curl --cacert certs/registry.luoruofeng.local.crt https://registry.luoruofeng.local:443/v2/_catalog

#login
docker login http://localhost:5000

#test 
docker pull alpine
docker tag alpine localhost:5000/alpine
docker push localhost:5000/alpine