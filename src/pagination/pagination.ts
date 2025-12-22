export class PAgeQuery {
  size: number;
  no: number;
}

export class Page<T> {
  pageNumber: number;
  pageSize: number;
  total: number;
  data: T[];
}

