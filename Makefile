.PHONY: help
help:

.PHONY: push
push:
	git tag -f v2 \
	&& git push -f origin main \
	&& git push -f origin v2

.PHONY: save
save:
	git cfx \
	&& GIT_SEQUENCE_EDITOR=: git rebase --root --autosquash

