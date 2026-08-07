PYTHON ?= $(shell test -x .venv/bin/python && echo .venv/bin/python || echo python3)

.PHONY: changelog
changelog:
	$(PYTHON) scripts/generate_release_notes.py
	cp "$$(printf '%s\n' CHANGELOG/v*.yml | grep -v '\.ru\.yml$$' | sort -V | tail -n1)" changelog.yaml

.PHONY: changelog-diff
changelog-diff:
	$(PYTHON) scripts/changelog_diff.py $(TAG)

.PHONY: update-base-images-versions
update-base-images-versions:
	##~ Options: version=vMAJOR.MINOR.PATCH
	cd .werf && curl --fail -sSLO https://fox.flant.com/api/v4/projects/deckhouse%2Fcontainer-base%2Fbase-images/packages/generic/base_images/${version}/base_images.yml
