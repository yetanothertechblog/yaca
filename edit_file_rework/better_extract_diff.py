#!/usr/bin/env python3

import json
import re
from pathlib import Path

def extract_diff_comprehensive(input_file, output_file):
    """
    Comprehensive extraction of diff content from Anthropic streaming JSON response.
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
    
    print(f"Extracted {len(all_json_parts)} JSON parts")
    
    # Combine all parts and clean up
    full_json = ''.join(all_json_parts)
    print(f"Combined JSON length: {len(full_json)}")
    
    # Clean up the JSON content
    full_json = full_json.replace('\\n', '\n')
    full_json = full_json.replace('\\"', '"')
    full_json = full_json.replace('\\\\', '\\')
    
    # Remove any control characters that might cause parsing issues
    full_json = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', full_json)
    
    # Try to reconstruct a complete JSON object
    # Look for the start and end of the JSON structure
    json_start = full_json.find('{')
    json_end = full_json.rfind('}')
    
    if json_start != -1 and json_end != -1 and json_end > json_start:
        reconstructed_json = full_json[json_start:json_end + 1]
        print(f"Reconstructed JSON length: {len(reconstructed_json)}")
        
        # Try to parse the reconstructed JSON
        try:
            diff_data = json.loads(reconstructed_json)
            if 'content' in diff_data:
                diff_content = diff_data['content']
                print("Successfully parsed reconstructed JSON")
                write_diff_to_file(diff_content, output_file)
                return True
        except json.JSONDecodeError:
            print("Reconstructed JSON parsing failed")
    
    # Try to extract the Go code directly from the combined content
    # Look for package declaration and the entire Go code
    go_pattern = r'package conversation[\s\S]*?(?=type\s+\w+|func\s+\w+|const\s+\w+|var\s+\w+|$)'
    go_match = re.search(go_pattern, full_json)
    
    if go_match:
        diff_content = go_match.group(0)
        print("Successfully extracted Go code pattern")
        write_diff_to_file(diff_content, output_file)
        return True
    
    # Try to find content between content quotes
    content_pattern = r'"content":\s*"([^"]*)"'
    content_matches = re.findall(content_pattern, full_json)
    
    if content_matches:
        # Take the longest content match
        diff_content = max(content_matches, key=len)
        print(f"Found {len(content_matches)} content matches, longest is {len(diff_content)} chars")
        write_diff_to_file(diff_content, output_file)
        return True
    
    # Last resort: write the full JSON content for debugging
    print("All extraction methods failed, writing full JSON for debugging")
    with open('debug_full_json.json', 'w') as f:
        f.write(full_json)
    
    return False

def write_diff_to_file(content, output_file):
    """Write diff content to file with proper formatting"""
    # Fix common formatting issues
    content = content.replace('\\t', '    ')  # Replace tabs with spaces
    content = content.replace('\\n', '\n')   # Fix escaped newlines
    content = content.replace('\\"', '"')    # Fix escaped quotes
    
    with open(output_file, 'w') as f:
        f.write(content)
    
    print(f"Successfully wrote diff to: {output_file}")
    print(f"Content length: {len(content)} characters")
    
    # Show first few lines
    lines = content.split('\n')[:5]
    print("First 5 lines:")
    for i, line in enumerate(lines):
        print(f"  {i+1}: {line}")

if __name__ == "__main__":
    input_file = "anthropic_edit_response.json"
    output_file = "single_diff.diff"
    
    if not Path(input_file).exists():
        print(f"Error: Input file '{input_file}' not found")
        exit(1)
        
    success = extract_diff_comprehensive(input_file, output_file)
    exit(0 if success else 1)