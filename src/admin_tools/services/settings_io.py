from pathlib import Path

import yaml

from admin_tools.models.overrides import RepoOverrideFile
from admin_tools.models.settings import OrgSettings


def load_org_settings(settings_path: Path) -> OrgSettings:
    settings_file = settings_path / "settings.yml"
    if not settings_file.exists():
        raise FileNotFoundError(f"Settings file not found: {settings_file}")
    with open(settings_file) as f:
        data = yaml.safe_load(f)
    return OrgSettings.model_validate(data or {})


_QUOTED_FIELDS = {"description", "homepage"}


def _format_value(key: str, value: str | bool) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if key in _QUOTED_FIELDS:
        return f'"{value}"'
    return value


def write_repo_override(override: RepoOverrideFile, repos_dir: Path) -> Path:
    repos_dir.mkdir(parents=True, exist_ok=True)
    outfile = repos_dir / f"{override.name}.yml"

    lines: list[str] = []
    header = (
        f"# {override.owner}/{override.name}"
        f" — overrides from {override.comment_source} defaults"
    )
    lines.append(header)
    lines.append("")

    if override.is_fork:
        lines.append("_fork: true")
        lines.append("")

    overrides = override.repository.model_dump(exclude_none=True)
    if not overrides:
        lines.append("repository: {}")
    else:
        lines.append("repository:")
        for key, value in overrides.items():
            lines.append(f"  {key}: {_format_value(key, value)}")

    content = "\n".join(lines) + "\n"
    outfile.write_text(content)
    return outfile


def list_existing_repo_files(repos_dir: Path) -> set[str]:
    if not repos_dir.exists():
        return set()
    return {f.stem for f in repos_dir.glob("*.yml")}
