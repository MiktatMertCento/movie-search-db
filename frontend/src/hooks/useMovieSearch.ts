import { useQuery } from '@tanstack/react-query';
import { fetchMovies } from '../api';

export const useMovieSearch = (query: string) => {
    return useQuery({
        queryKey: ['movies', query],
        queryFn: () => fetchMovies(query),
        enabled: query.length > 0,
        staleTime: 1000 * 60 * 5,
        retry: 1,
    });
};