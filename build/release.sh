template_image="dynatrace-oneagent-operator:snapshot"
current_image="dynatrace-agent-operator:${TRAVIS_TAG}"
mkdir artefacts
sed "s/${template_image}/${current_image}/g" deploy/kubernetes.yaml > artefacts/kubernetes.yaml
sed "s/docker.io\/dynatrace\/${template_image}/registry.connect.redhat.com\/dynatrace\/${current_image}/g" deploy/openshift.yaml > artefacts/openshift.yaml
