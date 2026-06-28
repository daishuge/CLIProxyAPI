/**
 * CodeMirror 6 YAML editor with automatic dark/light theme switching.
 *
 * Dependencies (already in package.json):
 *   @uiw/react-codemirror, @codemirror/lang-yaml
 */
import * as React from "react";
import CodeMirror from "@uiw/react-codemirror";
import { yaml } from "@codemirror/lang-yaml";
import { useThemeStore } from "@/lib/theme";

export interface YamlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  className?: string;
}

const extensions = [yaml()];

/**
 * Stateless YAML editor. The parent owns the value; this component only
 * fires `onChange` on user edits. Theme is derived from the global store.
 */
export function YamlEditor({ value, onChange, readOnly = false, className }: YamlEditorProps) {
  const isDark = useThemeStore((s) => s.isDark);

  const handleChange = React.useCallback(
    (val: string) => {
      onChange?.(val);
    },
    [onChange],
  );

  return (
    <CodeMirror
      value={value}
      onChange={handleChange}
      extensions={extensions}
      theme={isDark ? "dark" : "light"}
      readOnly={readOnly}
      editable={!readOnly}
      className={className}
      minHeight="500px"
      basicSetup={{
        lineNumbers: true,
        foldGutter: true,
        highlightActiveLine: !readOnly,
        highlightSelectionMatches: true,
      }}
    />
  );
}
