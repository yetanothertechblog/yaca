#!/usr/bin/env python3

import json
import re
from pathlib import Path

def convert_anthropic_stream_to_diff(input_file, output_file):
    """
    Convert Anthropic streaming JSON response to a single diff file.
    
    Args:
        input_file: Path to the input JSON stream file
        output_file: Path to the output diff file
    """
    json_content = ""
    
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
                            json_content += partial_json
                            
                except json.JSONDecodeError:
                    # Skip malformed JSON lines
                    continue
    
    # Clean up the JSON content
    json_content = json_content.replace('\\n', '\n')
    json_content = json_content.replace('\\"', '"')
    json_content = json_content.replace('\\\\', '\\')
    
    # Remove any control characters that might cause parsing issues
    json_content = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', json_content)
    
    # Try to parse as JSON, but if that fails, try to extract content directly
    try:
        diff_data = json.loads(json_content)
        if 'content' in diff_data:
            diff_content = diff_data['content']
        else:
            raise ValueError("No 'content' field found")
    except json.JSONDecodeError:
        # If JSON parsing fails, try to extract content using regex
        # Look for content between quotes after "content":
        content_match = re.search(r'"content":\s*"([^"]*)"', json_content)
        if content_match:
            diff_content = content_match.group(1)
        else:
            print(f"Error: Could not extract content from JSON")
            print(f"JSON content: {json_content[:500]}...")
            return False
    
    # Write the diff content to output file
    with open(output_file, 'w') as f:
        f.write(diff_content)
        
    print(f"Successfully extracted diff content to: {output_file}")
    return True

if __name__ == "__main__":
    input_file = "anthropic_edit_response.json"
    output_file = "single_diff.diff"
    
    if not Path(input_file).exists():
        print(f"Error: Input file '{input_file}' not found")
        exit(1)
        
    success = convert_anthropic_stream_to_diff(input_file, output_file)
    exit(0 if success else 1)