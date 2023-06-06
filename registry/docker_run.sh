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

# TLS and Auth
docker run -d   -p 5000:5000   --restart=always   --name registry   -v "$(pwd)"/auth:/auth   -e "REGISTRY_AUTH=htpasswd"   -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm"   -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd   -v "$(pwd)"/certs:/certs   -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/ca_cert.pem   -e REGISTRY_HTTP_TLS_KEY=/certs/ca_key.pem -v "${PWD}"/registry_data:/var/lib/registry   registry:2
#login
docker login http://localhost:5000

#test 
docker pull alpine
docker tag alpine localhost:5000/alpine
docker push localhost:5000/alpine