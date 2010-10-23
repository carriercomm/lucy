/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 * 
 *     http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#define C_KINO_TESTTERMQUERY
#include "KinoSearch/Util/ToolSet.h"
#include <math.h>

#include "KinoSearch/Test.h"
#include "KinoSearch/Test/Search/TestTermQuery.h"
#include "KinoSearch/Test/TestUtils.h"
#include "KinoSearch/Search/TermQuery.h"

static void
test_Dump_Load_and_Equals(TestBatch *batch)
{
    TermQuery *query         = TestUtils_make_term_query("content", "foo");
    TermQuery *field_differs = TestUtils_make_term_query("stuff", "foo");
    TermQuery *term_differs  = TestUtils_make_term_query("content", "bar");
    TermQuery *boost_differs = TestUtils_make_term_query("content", "foo");
    Obj       *dump          = (Obj*)TermQuery_Dump(query);
    TermQuery *clone         = (TermQuery*)TermQuery_Load(term_differs, dump);

    TEST_FALSE(batch, TermQuery_Equals(query, (Obj*)field_differs),
        "Equals() false with different field");
    TEST_FALSE(batch, TermQuery_Equals(query, (Obj*)term_differs),
        "Equals() false with different term");
    TermQuery_Set_Boost(boost_differs, 0.5);
    TEST_FALSE(batch, TermQuery_Equals(query, (Obj*)boost_differs),
        "Equals() false with different boost");
    TEST_TRUE(batch, TermQuery_Equals(query, (Obj*)clone), 
        "Dump => Load round trip");

    DECREF(query);
    DECREF(term_differs);
    DECREF(field_differs);
    DECREF(boost_differs);
    DECREF(dump);
    DECREF(clone);
}

void
TestTermQuery_run_tests()
{
    TestBatch *batch = TestBatch_new(4);
    TestBatch_Plan(batch);
    test_Dump_Load_and_Equals(batch);
    DECREF(batch);
}


