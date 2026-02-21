#!/usr/bin/env python3

import json
import re
from pathlib import Path

def extract_diff_from_stream(input_file, output_file):
    """
    Extract diff content from Anthropic streaming JSON response.
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
    
    print(f"Extracted JSON content length: {len(json_content)}")
    
    # Clean up the JSON content
    json_content = json_content.replace('\\n', '\n')
    json_content = json_content.replace('\\"', '"')
    json_content = json_content.replace('\\\\', '\\')
    
    # Remove any control characters that might cause parsing issues
    json_content = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', json_content)
    
    # Try different approaches to extract the content
    
    # Approach 1: Try to parse as JSON
    try:
        diff_data = json.loads(json_content)
        if 'content' in diff_data:
            diff_content = diff_data['content']
            print("Successfully parsed JSON and extracted content")
            write_diff_to_file(diff_content, output_file)
            return True
    except json.JSONDecodeError:
        print("JSON parsing failed, trying alternative methods...")
    
    # Approach 2: Extract content using regex
    content_match = re.search(r'"content":\s*"([^"]*)"', json_content)
    if content_match:
        diff_content = content_match.group(1)
        print("Successfully extracted content using regex")
        write_diff_to_file(diff_content, output_file)
        return True
    
    # Approach 3: Try to find the Go code directly
    # Look for package declaration and import statements
    go_code_match = re.search(r'package conversation[\s\S]*?(?=type|$)', json_content)
    if go_code_match:
        diff_content = go_code_match.group(0)
        print("Successfully extracted Go code directly")
        write_diff_to_file(diff_content, output_file)
        return True
    
    # Approach 4: Look for any content between quotes after file_path
    file_path_match = re.search(r'"file_path":\s*"([^"]*)"[^}]*"content":\s*"([^"]*)"', json_content)
    if file_path_match:
        diff_content = file_path_match.group(2)
        print("Successfully extracted content from file_path pattern")
        write_diff_to_file(diff_content, output_file)
        return True
    
    print("Failed to extract content using all methods")
    print(f"JSON content preview: {json_content[:1000]}...")
    return False

def write_diff_to_file(content, output_file):
    """Write diff content to file with proper formatting"""
    # Fix common formatting issues
    content = content.replace('\t', '    ')  # Replace tabs with spaces
    content = content.replace('\\n', '\n')   # Fix escaped newlines
    content = content.replace('\\"', '"')    # Fix escaped quotes
    
    with open(output_file, 'w') as f:
        f.write(content)
    
    print(f"Successfully wrote diff to: {output_file}")
    print(f"Content length: {len(content)} characters")

if __name__ == "__main__":
    input_file = "anthropic_edit_response.json"
    output_file = "single_diff.diff"
    
    if not Path(input_file).exists():
        print(f"Error: Input file '{input_file}' not found")
        exit(1)
        
    success = extract_diff_from_stream(input_file, output_file)
    exit(0 if success else 1)