import axios from 'axios';
import type {Movie} from './types/schema.ts';

const api = axios.create({
    baseURL: 'http://localhost:8080/api',
});

export const fetchMovies = async (query: string): Promise<Movie[]> => {
    if (!query) return [];
    const { data } = await api.post<Movie[]>('/search', { query });
    return data;
};