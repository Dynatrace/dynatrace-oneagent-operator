echo "[!] Releasing..."
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash

template_image="dynatrace-oneagent-operator:snapshot"
current_image="dynatrace-oneagent-operator:${TRAVIS_TAG}"
mkdir artefacts

./kustomize build ./deploy/kubernetes -o kubernetes.yaml
./kustomize build ./deploy/openshift -o openshift.yaml

sed "s/${template_image}/${current_image}/g" kubernetes.yaml >artefacts/kubernetes.yaml
sed "s/docker.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" openshift.yaml >artefacts/openshift.yaml
