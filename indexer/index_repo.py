#!/usr/bin/env python3
"""Walk the workspace and emit a JSONL chunk index.

Each line of the output file is a JSON object:

    {
      "path": "relative/path/file.go",
      "lang": "go",
      "size": 1234,
      "start_line": 1,
      "end_line": 80,
      "text": "..."
    }

The indexer is read-only; it never modifies the workspace.
"""
from __future__ import annotations

import json
import os
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Iterator

WORKSPACE_DIR = Path(os.environ.get("WORKSPACE_DIR", "/workspace"))
INDEX_DIR = Path(os.environ.get("INDEX_DIR", "/data/index"))
OUTPUT = INDEX_DIR / "repo_index.jsonl"

IGNORED_DIRS = {
    ".git",
    "node_modules",
    "vendor",
    "dist",
    "build",
    ".venv",
    "__pycache__",
    ".next",
    ".cache",
    "target",
    ".idea",
    ".vscode",
}

EXTENSION_LANG = {
    ".go": "go",
    ".py": "python",
    ".js": "javascript",
    ".jsx": "javascript",
    ".ts": "typescript",
    ".tsx": "typescript",
    ".md": "markdown",
    ".yaml": "yaml",
    ".yml": "yaml",
    ".json": "json",
    ".ps1": "powershell",
    ".sh": "shell",
    ".rs": "rust",
    ".java": "java",
    ".kt": "kotlin",
    ".rb": "ruby",
    ".php": "php",
    ".c": "c",
    ".h": "c",
    ".cc": "cpp",
    ".cpp": "cpp",
    ".hpp": "cpp",
    ".sql": "sql",
}

SPECIAL_NAMES = {
    "Dockerfile": "dockerfile",
    "Makefile": "make",
    "go.mod": "go",
    "go.sum": "go",
    "requirements.txt": "python",
    "pyproject.toml": "python",
    "package.json": "javascript",
    "tsconfig.json": "typescript",
}

CHUNK_LINES = 80
CHUNK_OVERLAP = 10
MAX_FILE_BYTES = 1_000_000  # skip files larger than 1 MB


@dataclass
class Chunk:
    path: str
    lang: str
    size: int
    start_line: int
    end_line: int
    text: str

    def to_json(self) -> str:
        return json.dumps(
            {
                "path": self.path,
                "lang": self.lang,
                "size": self.size,
                "start_line": self.start_line,
                "end_line": self.end_line,
                "text": self.text,
            },
            ensure_ascii=False,
        )


def detect_language(path: Path) -> str | None:
    if path.name in SPECIAL_NAMES:
        return SPECIAL_NAMES[path.name]
    return EXTENSION_LANG.get(path.suffix.lower())


def iter_files(root: Path) -> Iterator[Path]:
    for current_root, dirs, files in os.walk(root):
        dirs[:] = [d for d in dirs if d not in IGNORED_DIRS]
        for name in files:
            yield Path(current_root, name)


def chunk_lines(lines: list[str]) -> Iterable[tuple[int, int, str]]:
    """Yield (start_line, end_line, text) tuples with line overlap."""
    if not lines:
        return
    n = len(lines)
    step = CHUNK_LINES - CHUNK_OVERLAP
    if step <= 0:
        step = CHUNK_LINES
    start = 0
    while start < n:
        end = min(start + CHUNK_LINES, n)
        text = "".join(lines[start:end])
        yield (start + 1, end, text)
        if end == n:
            break
        start += step


def index_file(path: Path, root: Path) -> Iterator[Chunk]:
    try:
        size = path.stat().st_size
    except OSError:
        return
    if size > MAX_FILE_BYTES:
        return
    lang = detect_language(path)
    if lang is None:
        return
    try:
        with path.open("r", encoding="utf-8", errors="replace") as f:
            lines = f.readlines()
    except OSError:
        return
    rel = path.relative_to(root).as_posix()
    for start, end, text in chunk_lines(lines):
        yield Chunk(
            path=rel,
            lang=lang,
            size=size,
            start_line=start,
            end_line=end,
            text=text,
        )


def main() -> int:
    if not WORKSPACE_DIR.is_dir():
        print(f"workspace directory missing: {WORKSPACE_DIR}", file=sys.stderr)
        return 1
    INDEX_DIR.mkdir(parents=True, exist_ok=True)

    files_indexed = 0
    chunks_written = 0
    with OUTPUT.open("w", encoding="utf-8") as out:
        for file_path in iter_files(WORKSPACE_DIR):
            wrote_chunk = False
            for chunk in index_file(file_path, WORKSPACE_DIR):
                out.write(chunk.to_json())
                out.write("\n")
                chunks_written += 1
                wrote_chunk = True
            if wrote_chunk:
                files_indexed += 1

    print(
        f"indexed {files_indexed} files into {chunks_written} chunks -> {OUTPUT}",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
