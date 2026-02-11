# Hedgerules

MAKEFLAGS += --warn-undefined-variables
.SHELLFLAGS := -euc
.SUFFIXES:

GO_SRC := $(shell find hedgerules -name '*.go' -o -name '*.js') hedgerules/go.mod hedgerules/go.sum
HUGO_SRC := $(shell find www/config www/content themes -type f 2>/dev/null)

# Show a nice table of Make targets.
.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Go ---

.PHONY: gofmt
gofmt: ## Run gofmt on all Go source files.
	cd hedgerules && gofmt -w .

.PHONY: govet
govet: ## Run go vet on all Go source files.
	cd hedgerules && go vet ./...

bin/hedgerules: $(GO_SRC) | gofmt govet ## Build the hedgerules binary.
	mkdir -p $(dir $@)
	cd hedgerules && go build -o $(abspath $@) ./cmd/hedgerules

# --- Hugo ---

www/public/production/.sentinel: $(HUGO_SRC)
	cd www && hugo --environment production
	@touch $@

.PHONY: hugo-build
hugo-build: www/public/production/.sentinel ## Build the Hugo site for production.

.PHONY: hugo-serve
hugo-serve: ## Serve the Hugo site locally with live reload.
	cd www && hugo server --buildDrafts

.PHONY: hugo-deploy
hugo-deploy: www/public/production/.sentinel ## Build the Hugo site for production
	cd www && hugo deploy --environment production --target hedgerules-micahrl-com-s3

# --- Hedgerules ---

.PHONY: hedgerules-deploy
hedgerules-deploy: bin/hedgerules www/public/production/.sentinel ## Deploy the hedgerules application to production.
	bin/hedgerules deploy

# --- CloudFormation ---

CFN_REGION := us-east-1
CFN_DIST := infra/prod/dist

.PHONY: cfn-base
cfn-base: $(CFN_DIST)/base.outputs.json ## Deploy the base infrastructure CloudFormation stack (stage 1).

.PHONY: cfn-distribution
cfn-distribution: $(CFN_DIST)/distribution.outputs.json ## Deploy the distribution CloudFormation stack (stage 4).

$(CFN_DIST)/base.outputs.json: infra/prod/base.cfn.yml
	mkdir -p $(dir $@)
	aws cloudformation deploy \
		--region $(CFN_REGION) \
		--template-file $< \
		--stack-name hedgerules-base-prod \
		--capabilities CAPABILITY_NAMED_IAM \
		--no-fail-on-empty-changeset
	aws cloudformation describe-stacks \
		--region $(CFN_REGION) \
		--stack-name hedgerules-base-prod \
		--query 'Stacks[0].Outputs' \
		--output json \
		| jq 'map({(.OutputKey): .OutputValue}) | add' > $@

$(CFN_DIST)/distribution.outputs.json: BUCKET = $(shell jq -r '.ContentBucketName' $(CFN_DIST)/base.outputs.json)
$(CFN_DIST)/distribution.outputs.json: BUCKET_DOMAIN = $(shell jq -r '.ContentBucketRegionalDomainName' $(CFN_DIST)/base.outputs.json)
$(CFN_DIST)/distribution.outputs.json: OAC_ID = $(shell jq -r '.OriginAccessControlId' $(CFN_DIST)/base.outputs.json)
$(CFN_DIST)/distribution.outputs.json: CERT_ARN = $(shell jq -r '.CertificateArn' $(CFN_DIST)/base.outputs.json)
$(CFN_DIST)/distribution.outputs.json: infra/prod/distribution.cfn.yml $(CFN_DIST)/base.outputs.json
	aws cloudformation deploy \
		--region $(CFN_REGION) \
		--template-file $< \
		--stack-name hedgerules-distrib-prod \
		--no-fail-on-empty-changeset \
		--parameter-overrides \
			ContentBucketName=$(BUCKET) \
			ContentBucketRegionalDomainName=$(BUCKET_DOMAIN) \
			OriginAccessControlId=$(OAC_ID) \
			CertificateArn=$(CERT_ARN) \
			RequestFunctionName=hedgerules-base-prod-viewer-request \
			ResponseFunctionName=hedgerules-base-prod-viewer-response
	aws cloudformation describe-stacks \
		--region $(CFN_REGION) \
		--stack-name hedgerules-distrib-prod \
		--query 'Stacks[0].Outputs' \
		--output json \
		| tee $@

$(CFN_DIST)/cert-validation.json: CERT_ARN = $(shell jq -r '.CertificateArn' $(CFN_DIST)/base.outputs.json)
$(CFN_DIST)/cert-validation.json: $(CFN_DIST)/base.outputs.json
	aws acm describe-certificate \
		--region $(CFN_REGION) \
		--certificate-arn $(CERT_ARN) \
		--query 'Certificate.DomainValidationOptions[0].ResourceRecord' \
		| tee $@

.PHONY: cfn-cert-validation
cfn-cert-validation: $(CFN_DIST)/cert-validation.json ## Show the ACM certificate validation DNS record.
	@cat $(CFN_DIST)/cert-validation.json

.PHONY: cfn-cname
cfn-cname: $(CFN_DIST)/distribution.outputs.json ## Show the CNAME target for hedgerules.micahrl.com.
	@jq -r '.DistributionDomainName' $(CFN_DIST)/distribution.outputs.json

# --- Clean ---

clean: ## Clean up all build artifacts.
	rm -rf hedgerules/bin www/public/production $(CFN_DIST)

# --- Deploy everything ---

.PHONY: deploy
deploy: cfn-base cfn-distribution hugo-build hedgerules-deploy hugo-deploy ## Deploy all infrastructure and applications to production.
