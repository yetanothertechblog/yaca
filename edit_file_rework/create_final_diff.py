#!/usr/bin/env python3

import json
import re
from pathlib import Path

def create_proper_diff(input_file, output_file):
    """
    Create a proper diff file from the extracted content.
    """
    all_json_parts = []
    
    with open(input_file, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
                
            # Parse the line to extract the JSON data
            if line.startswith('data: '):
                json_str = line[6:]  # Remove "data: " prefix
                try:
                    data = json.loads(json_str)
                    
                    # Extract partial JSON chunks from input_json_delta events
                    if (data.get('type') == 'content_block_delta' and 
                        data.get('index') == 2 and
                        'delta' in data and
                        data['delta'].get('type') == 'input_json_delta'):
                        
                        partial_json = data['delta'].get('partial_json', '')
                        if partial_json:
                            all_json_parts.append(partial_json)
                            
                except json.JSONDecodeError:
                    # Skip malformed JSON lines
                    continue
    
    # Combine all parts and clean up
    full_json = ''.join(all_json_parts)
    full_json = full_json.replace('\\n', '\n')
    full_json = full_json.replace('\\"', '"')
    full_json = full_json.replace('\\\\', '\\')
    full_json = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', full_json)
    
    # Extract Go code using pattern matching
    go_pattern = r'package conversation[\s\S]*?(?=type\s+\w+|func\s+\w+|const\s+\w+|var\s+\w+|$)'
    go_match = re.search(go_pattern, full_json)
    
    if go_match:
        diff_content = go_match.group(0)
        
        # Clean up the Go code formatting
        diff_content = re.sub(r'import\s*\(\s*"([^"]*)"', r'import (\n\t"\1"', diff_content)
        diff_content = re.sub(r'"([^"]*)"', r'"\1"\n\t', diff_content)
        
        # Create a proper diff format
        diff_lines = [
            "diff --git a/conversation/conversation.go b/conversation/conversation.go",
            "index 0000000..0000000",
            "--- a/conversation/conversation.go",
            "+++ b/conversation/conversation.go",
            "@@ -0,0 +1,{} @@".format(len(diff_content.split('\n'))),
            ""
        ]
        
        # Add the Go code with proper indentation
        for line in diff_content.split('\n'):
            if line.strip():
                diff_lines.append('+{}'.format(line))
            else:
                diff_lines.append('+')
        
        # Write the diff
        with open(output_file, 'w') as f:
            f.write('\n'.join(diff_lines))
        
        print(f"Successfully created proper diff file: {output_file}")
        print(f"Diff contains {len(diff_lines)} lines")
        return True
    
    return False

if __name__ == "__main__":
    input_file = "anthropic_edit_response.json"
    output_file = "final_diff.diff"
    
    if not Path(input_file).exists():
        print(f"Error: Input file '{input_file}' not found")
        exit(1)
        
    success = create_proper_diff(input_file, output_file)
    exit(0 if success else 1)