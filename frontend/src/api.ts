import axios from 'axios';
import type {Movie} from './types/schema.ts';

const api = axios.create({
    baseURL: 'https://searchapi.miktatmert.dev/api',
});

export const fetchMovies = async (query: string): Promise<Movie[]> => {
    if (!query) return [];
    const { data } = await api.post<Movie[]>('/search', { query });
    return data;
};