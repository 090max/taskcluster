import json

from taskgraph.transforms.base import TransformSequence

transforms = TransformSequence()


@transforms.add
def taskcluster_images(config, jobs):
    with open('package.json', 'r') as pkg:
        with open('.go-version', 'r') as goversion:
            node_version = json.load(pkg)["engines"]["node"]
            go_version = goversion.read()
            pg_version = 11
            for job in jobs:
                image = job["worker"]["docker-image"]
                if isinstance(image, dict) and image.keys()[0] == "taskcluster":
                    repo = image["taskcluster"]
                    if (repo == "node-and-go"):
                        image = "taskcluster/node-and-go:node{node_version}-{go_version}"
                    elif (repo == "node-and-postgres"):
                        image = "taskcluster/node-and-postgres:node{node_version}-pg{pg_version}"
                    elif (repo == "browser-test"):
                        image = "taskcluster/browser-test:node{node_version}"

                    job["worker"]["docker-image"] = image.format(
                        node_version=node_version,
                        go_version=go_version,
                        pg_version=pg_version
                    ).strip()

                yield job


@transforms.add
def add_dw_env(config, jobs):
    for job in jobs:
        env = job["worker"].setdefault("env", {})
        env["GITHUB_REPO_URL"] = config.params["head_repository"]
        env["GITHUB_BRANCH"] = config.params["head_ref"]
        env["GITHUB_SHA"] = config.params["head_rev"]
        yield job