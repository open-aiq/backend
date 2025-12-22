import { Injectable } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { Prisma, User } from 'generated/prisma';
import { DefaultArgs } from 'generated/prisma/runtime/library';
import { Page, PAgeQuery } from '../pagination/pagination';

export type UserModel = User;
export type UserModelPage = Page<UserModel>;

@Injectable()
export class UsersService {
  private readonly userDao: Prisma.UserDelegate<
    DefaultArgs,
    Prisma.PrismaClientOptions
  >;
  constructor(private readonly prismaService: PrismaService) {
    this.userDao = prismaService.user;
  }

  async findPaginated(pageQuery: PAgeQuery): Promise<UserModelPage> {
    const skip = (pageQuery.no - 1) * pageQuery.size;

    const [users, totalCount] = await this.prismaService.$transaction([
      this.userDao.findMany({
        skip: skip,
        take: pageQuery.size,
        orderBy: { createdAt: 'desc' },
      }),
      this.userDao.count(), // Get total for calculating total pages
    ]);

    const totalPages = totalCount / pageQuery.size;

    return {
      data: users,
      pageNumber: pageQuery.no,
      pageSize: pageQuery.size,
      total: totalPages,
    };
  }
}
