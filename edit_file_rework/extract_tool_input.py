#!/usr/bin/env python3

import json
import re
from pathlib import Path

def extract_tool_input():
    """Extract the complete tool input from the streaming response"""
    json_content = ""
    
    with open('anthropic_edit_response.json', 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
                
            if line.startswith('data: '):
                json_str = line[6:]
                try:
                    data = json.loads(json_str)
                    
                    if (data.get('type') == 'content_block_delta' and 
                        data.get('index') == 2 and
                        'delta' in data and
                        data['delta'].get('type') == 'input_json_delta'):
                        
                        partial_json = data['delta'].get('partial_json', '')
                        if partial_json:
                            json_content += partial_json
                            
                except json.JSONDecodeError:
                    continue
    
    # Clean up and reconstruct
    json_content = json_content.replace('\\n', '\n')
    json_content = json_content.replace('\\"', '"')
    json_content = json_content.replace('\\\\', '\\')
    json_content = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', json_content)
    
    # Find JSON start and end
    json_start = json_content.find('{')
    json_end = json_content.rfind('}')
    
    if json_start != -1 and json_end != -1:
        reconstructed_json = json_content[json_start:json_end + 1]
        print("Reconstructed Tool Input:")
        print(reconstructed_json)
        
        try:
            parsed = json.loads(reconstructed_json)
            print("\nParsed Tool Input:")
            print(json.dumps(parsed, indent=2))
        except json.JSONDecodeError:
            print("Could not parse as JSON")
    else:
        print("Could not find JSON boundaries")
        print(f"JSON content: {json_content[:1000]}...")

if __name__ == "__main__":
    extract_tool_input()