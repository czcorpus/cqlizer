"""
Syntax check for DQD queries, using the reference grammar
in dqd/dqd_grammar.lark (the same grammar LCP uses).

The grammar is not part of this repository (license reasons), it must
be obtained from https://github.com/liri-uzh/lcp/blob/master/parser/dqd_grammar.lark
and stored as dqd/dqd_grammar.lark.

usage:
    python scripts/dqd_check.py [--tree] [file ...]

With no files, the query is read from stdin. Exits with a non-zero
status if any of the queries fails to parse.
"""

import argparse
import os
import sys

from lark import Lark
from lark.exceptions import LarkError
from lark.indenter import Indenter

GRAMMAR_PATH = os.path.join(os.path.dirname(__file__), '..', 'dqd', 'dqd_grammar.lark')


class TreeIndenter(Indenter):
    """
    Turns indentation into _INDENT/_DEDENT tokens declared (but not defined)
    by the grammar. Mirrors the postlexer used by LCP's DQD parser.
    """
    NL_type = '_NL'
    OPEN_PAREN_types = []
    CLOSE_PAREN_types = []
    INDENT_type = '_INDENT'
    DEDENT_type = '_DEDENT'
    tab_len = 4


def create_parser():
    with open(GRAMMAR_PATH) as fr:
        # The grammar is not LALR(1) (e.g. label/entity/reference rules are identical),
        # so Earley is needed. The postlexer requires the 'basic' (non-dynamic) lexer.
        return Lark(fr.read(), parser='earley', lexer='basic', postlex=TreeIndenter(), start='top')


def check(parser, name, query, show_tree):
    # the grammar expects each statement (incl. the last one) to be terminated by a newline
    if not query.endswith('\n'):
        query += '\n'
    try:
        tree = parser.parse(query)
    except LarkError as ex:
        print(f'{name}: ERROR\n{ex}', file=sys.stderr)
        return False
    print(f'{name}: OK')
    if show_tree:
        print(tree.pretty())
    return True


if __name__ == '__main__':
    argp = argparse.ArgumentParser(description='Check syntax of DQD queries')
    argp.add_argument('files', nargs='*', help='files with DQD queries (one query per file); stdin if empty')
    argp.add_argument('--tree', action='store_true', help='print the parse tree')
    args = argp.parse_args()

    if not os.path.isfile(GRAMMAR_PATH):
        print(f'DQD grammar not found at {os.path.normpath(GRAMMAR_PATH)} (see the script docs)', file=sys.stderr)
        sys.exit(2)
    parser = create_parser()
    if args.files:
        ok = True
        for path in args.files:
            with open(path) as fr:
                ok = check(parser, path, fr.read(), args.tree) and ok
    else:
        ok = check(parser, '<stdin>', sys.stdin.read(), args.tree)
    sys.exit(0 if ok else 1)
