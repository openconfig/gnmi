package com.cisco.commons.networking;

import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;

import org.apache.commons.lang3.StringUtils;

import gnmi.Gnmi.Path;
import gnmi.Gnmi.Path.Builder;
import gnmi.Gnmi.PathElem;

/**
 * GNMI Path parser.
 * Following:
 * https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-strings.md#stringified-path-examples
 * 
 * This is intended for configuration flow, not data processing flow. As such, simplicity is preferred:
 * "String encoding MUST NOT be used within the protocol, but provides means to simplify human interaction with 
 * gNMI paths - for example, in device or daemon configuration"
 * 
 * @author Liran Mendelovich
 * 
 * Copyright 2019 Cisco Systems
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
public class GNMIUtils {
	
	private static char TOKEN_SEPARATOR = '/';
	
	private GNMIUtils() {
		
	}
	
	public static Path parseGNMIPathStr(String gnmiPathStr) {
		try {
			return parseGNMIPathStrHelper(gnmiPathStr);
		} catch (Exception e) {
			throw new IllegalArgumentException("Error parsing GNMI path string: " + gnmiPathStr, e);
		}
	}

	private static Path parseGNMIPathStrHelper(String gnmiPathStr) {
		List<String> tokens = getParts(gnmiPathStr);
        Builder pathBuilder = Path.newBuilder();
        boolean isFirst = true;
        for (String token: tokens) {
        	boolean isOrigin = false;
        	if (isFirst && !StringUtils.isBlank(token)) {
        		pathBuilder.setOrigin(token);
        		isOrigin = true;
        	}
        	isFirst = false;
            if(!StringUtils.isBlank(token) && !isOrigin) {
                String elem = null;
                Map<String,String> map = new HashMap<>();
                int kvsStart = token.indexOf("[");
                if(kvsStart >= 0){
					elem = token.substring(0, kvsStart);
                    String kvsStr = token.substring(kvsStart + 1, token.length() - 1);
                    String[] kvs = kvsStr.split("\\]\\[");
                    for (String kv : kvs) {
                    	String[] kpArr = kv.split("=");
                        map.put(kpArr[0], kpArr[1]);
    				}
                }else{
                    elem = token;
                }
                PathElem.Builder elemBuiler = PathElem.newBuilder().setName(elem);
                Set<Entry<String, String>> entries = map.entrySet();
                for (Entry<String, String> entry: entries) {
                	elemBuiler.putKey(entry.getKey(), entry.getValue());
				}
                pathBuilder.addElem(elemBuiler.build());
            }
        }
        return pathBuilder.build();
	}

	private static List<String> getParts(String gnmiPathStr) {
		List<String> parts = new LinkedList<>();
		int size = gnmiPathStr.length();
		if (size == 0) {
			return parts;
		}
		StringBuilder part = new StringBuilder();
		boolean isValue = false;
		char prevChar = gnmiPathStr.charAt(0);
		for (int i = 0; i < size; i++){
			if (i > 0) {
				prevChar = gnmiPathStr.charAt(i-1);
			}
		    char c = gnmiPathStr.charAt(i);
		    if (c == '[' && prevChar != '\\' ) {
		    		isValue = true;
		    } else if (c == ']' && prevChar != '\\' ) {
	    		isValue = false;
		    }
		    if (c == TOKEN_SEPARATOR && !isValue) {
		    	parts.add(part.toString());
		    	part = new StringBuilder();
		    } else {
		    	part.append(c);
		    }
		}
		if (part.length() > 0) {
			parts.add(part.toString());
		}
		return parts;
	}
	
	public static String toString(Path gnmiPath) {
		StringBuilder stringBuilder = new StringBuilder();
		stringBuilder.append(gnmiPath.getOrigin());
		List<PathElem> pathElements = gnmiPath.getElemList();
		for (PathElem pathElem: pathElements) {
			stringBuilder.append(TOKEN_SEPARATOR);
			stringBuilder.append(pathElem.getName());
			Set<Entry<String, String>> entries = pathElem.getKeyMap().entrySet();
			for (Entry<String, String> entry: entries) {
				stringBuilder.append("[");
				stringBuilder.append(entry.getKey());
				stringBuilder.append("=");
				stringBuilder.append(entry.getValue());
				stringBuilder.append("]");
			}
		}
		return stringBuilder.toString();
	}
}
