// frontend/src/lib/api/backlog.ts

import { BacklogItem, BacklogMetrics, BacklogItemFilters } from '@/types/backlog';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Error handler helper
const handleApiError = async (response: Response) => {
  if (!response.ok) {
    const errorText = await response.text();
    let errorMessage: string;
    
    try {
      const errorJson = JSON.parse(errorText);
      errorMessage = errorJson.message || errorJson.error || `Server error: ${response.status}`;
    } catch {
      errorMessage = errorText || `Server error: ${response.status}`;
    }
    
    throw new Error(errorMessage);
  }
  
  return response.json();
};

// Fetch backlog items with filters
export const fetchBacklogItems = async (filters: BacklogItemFilters = {}) => {
  // Build query parameters
  const params = new URLSearchParams();
  
  if (filters.type) params.append('types', filters.type);
  if (filters.status) params.append('statuses', filters.status);
  if (filters.assignee) params.append('assignee', filters.assignee);
  if (filters.searchQuery) params.append('search_query', filters.searchQuery);
  if (filters.parentId) params.append('parent_id', filters.parentId);
  if (filters.pageSize) params.append('page_size', filters.pageSize.toString());
  if (filters.pageToken) params.append('page_token', filters.pageToken.toString());
  
  const url = `${API_BASE_URL}/api/v1/backlog/items?${params.toString()}`;
  
  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error fetching backlog items:', error);
    throw error;
  }
};

// Create a new backlog item
export const createBacklogItem = async (item: Omit<BacklogItem, 'id' | 'createdAt' | 'updatedAt' | 'externalIds'>) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify(item),
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error creating backlog item:', error);
    throw error;
  }
};

// Update an existing backlog item
export const updateBacklogItem = async (item: Partial<BacklogItem> & { id: string }) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/${item.id}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify(item),
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error updating backlog item:', error);
    throw error;
  }
};

// Delete a backlog item
export const deleteBacklogItem = async (id: string) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/${id}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error deleting backlog item:', error);
    throw error;
  }
};

// Fetch a single backlog item by ID
export const fetchBacklogItem = async (id: string) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/${id}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error fetching backlog item:', error);
    throw error;
  }
};

// Fetch children of a backlog item
export const fetchChildrenItems = async (parentId: string) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/${parentId}/children`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error fetching children items:', error);
    throw error;
  }
};

// Reorder backlog items
export const reorderBacklogItems = async (items: { id: string; priority: number }[]) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/reorder`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify({ items }),
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error reordering backlog items:', error);
    throw error;
  }
};

// Set external ID for a backlog item
export const setExternalId = async (id: string, system: string, externalId: string) => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/items/${id}/external-id`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: JSON.stringify({ system, externalId }),
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error setting external ID:', error);
    throw error;
  }
};

// Fetch backlog metrics
export const fetchBacklogMetrics = async (): Promise<BacklogMetrics> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/backlog/metrics`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    
    return handleApiError(response);
  } catch (error) {
    console.error('Error fetching backlog metrics:', error);
    throw error;
  }
};

// frontend/src/hooks/useBacklogFilters.ts

import { useCallback } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { BacklogItemFilters } from '@/types/backlog';

export function useBacklogFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  
  // Get current filter values from URL
  const filters: BacklogItemFilters = {
    type: searchParams.get('type') || '',
    status: searchParams.get('status') || '',
    assignee: searchParams.get('assignee') || '',
    searchQuery: searchParams.get('search') || '',
    parentId: searchParams.get('parentId') || '',
  };
  
  // Helper to create new search params
  const createQueryString = useCallback(
    (name: string, value: string) => {
      const params = new URLSearchParams(searchParams.toString());
      if (value) {
        params.set(name, value);
      } else {
        params.delete(name);
      }
      return params.toString();
    },
    [searchParams]
  );
  
  // Filter setter functions
  const setTypeFilter = useCallback(
    (value: string) => {
      router.push(`${pathname}?${createQueryString('type', value)}`);
    },
    [pathname, createQueryString, router]
  );
  
  const setStatusFilter = useCallback(
    (value: string) => {
      router.push(`${pathname}?${createQueryString('status', value)}`);
    },
    [pathname, createQueryString, router]
  );
  
  const setAssigneeFilter = useCallback(
    (value: string) => {
      router.push(`${pathname}?${createQueryString('assignee', value)}`);
    },
    [pathname, createQueryString, router]
  );
  
  const setSearchQuery = useCallback(
    (value: string) => {
      router.push(`${pathname}?${createQueryString('search', value)}`);
    },
    [pathname, createQueryString, router]
  );
  
  const setParentIdFilter = useCallback(
    (value: string) => {
      router.push(`${pathname}?${createQueryString('parentId', value)}`);
    },
    [pathname, createQueryString, router]
  );
  
  // Clear all filters
  const clearFilters = useCallback(() => {
    router.push(pathname);
  }, [pathname, router]);
  
  // Check if there are any active filters
  const hasActiveFilters = Object.values(filters).some(Boolean);
  
  return {
    filters,
    setTypeFilter,
    setStatusFilter,
    setAssigneeFilter,
    setSearchQuery,
    setParentIdFilter,
    clearFilters,
    hasActiveFilters,
  };
}

// frontend/src/types/backlog.ts

export interface BacklogItem {
  id: string;
  type: string;
  parentId?: string;
  title: string;
  description: string;
  storyPoints: number;
  status: string;
  priority: number;
  assignee: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  externalIds: Record<string, string>;
}

export interface BacklogMetrics {
  totalItems: number;
  epicCount: number;
  featureCount: number;
  storyCount: number;
  averageAge: number;
  wipCount: number;
  leadTimeDays: number;
  throughputLast30Days: number;
  icebergRatio: number;
  healthStatus: string;
}

export interface BacklogItemFilters {
  type?: string;
  status?: string;
  assignee?: string;
  searchQuery?: string;
  parentId?: string;
  pageSize?: number;
  pageToken?: number;
  sortBy?: string;
  sortOrder?: string;
}

export interface ListItemsResponse {
  items: BacklogItem[];
  totalCount: number;
  nextPageToken: number;
}

export interface ReorderItem {
  id: string;
  priority: number;
}

// frontend/src/app/api/v1/backlog/[...path]/route.ts

/**
 * This file is a proxy to the backend API.
 * It allows us to avoid CORS issues during development.
 */

import { NextRequest, NextResponse } from 'next/server';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080';

export async function GET(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const { searchParams } = new URL(request.url);
  
  const url = `${API_BASE_URL}/api/v1/backlog/${path}${
    searchParams.toString() ? `?${searchParams.toString()}` : ''
  }`;
  
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
  });
  
  const data = await response.json();
  
  return NextResponse.json(data, {
    status: response.status,
  });
}

export async function POST(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const body = await request.json();
  
  const response = await fetch(`${API_BASE_URL}/api/v1/backlog/${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    body: JSON.stringify(body),
  });
  
  const data = await response.json();
  
  return NextResponse.json(data, {
    status: response.status,
  });
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const body = await request.json();
  
  const response = await fetch(`${API_BASE_URL}/api/v1/backlog/${path}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    body: JSON.stringify(body),
  });
  
  const data = await response.json();
  
  return NextResponse.json(data, {
    status: response.status,
  });
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  
  const response = await fetch(`${API_BASE_URL}/api/v1/backlog/${path}`, {
    method: 'DELETE',
    headers: {
      'Accept': 'application/json',
    },
  });
  
  if (response.status === 204) {
    return new NextResponse(null, { status: 204 });
  }
  
  const data = await response.json();
  
  return NextResponse.json(data, {
    status: response.status,
  });
}
